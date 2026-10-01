
#include <windows.h>
#include <shobjidl.h>
#include <shlobj.h>

#include <atomic>
#include <string>

namespace {

const CLSID kUpitExplorerCommandClsid = {
    0x5c4a83f8,
    0x4d35,
    0x4c0c,
    {0xb2, 0x5b, 0x2a, 0x57, 0xfd, 0xda, 0x8b, 0xb4},
};

const GUID kUpitExplorerCommandCanonicalName = {
    0x0c2b8f8f,
    0x9cc5,
    0x4b36,
    {0x9c, 0x88, 0x9f, 0xc1, 0x4e, 0x68, 0xea, 0x8e},
};

std::atomic<ULONG> g_serverLocks = 0;
HMODULE g_module = nullptr;

HRESULT copyString(const std::wstring& value, LPWSTR* result) {
    if (result == nullptr) {
        return E_POINTER;
    }
    const size_t bytes = (value.size() + 1) * sizeof(wchar_t);
    auto* copy = static_cast<LPWSTR>(CoTaskMemAlloc(bytes));
    if (copy == nullptr) {
        return E_OUTOFMEMORY;
    }
    memcpy(copy, value.c_str(), bytes);
    *result = copy;
    return S_OK;
}

std::wstring modulePath() {
    wchar_t path[MAX_PATH] = {};
    const DWORD length = GetModuleFileNameW(g_module, path, ARRAYSIZE(path));
    if (length == 0 || length >= ARRAYSIZE(path)) {
        return {};
    }
    return std::wstring(path, length);
}

std::wstring siblingPath(const std::wstring& name) {
    const std::wstring module = modulePath();
    const std::wstring::size_type slash = module.find_last_of(L"\\/");
    if (slash == std::wstring::npos) {
        return name;
    }
    return module.substr(0, slash + 1) + name;
}

std::wstring quoteCommandLineArgument(const std::wstring& value) {
    std::wstring result;
    result.reserve(value.size() + 2);
    result.push_back(L'"');
    size_t backslashes = 0;
    for (const wchar_t character : value) {
        if (character == L'\\') {
            ++backslashes;
            continue;
        }
        if (character == L'"') {
            result.append(backslashes * 2 + 1, L'\\');
            result.push_back(character);
            backslashes = 0;
            continue;
        }
        result.append(backslashes, L'\\');
        result.push_back(character);
        backslashes = 0;
    }
    result.append(backslashes * 2, L'\\');
    result.push_back(L'"');
    return result;
}

bool isExactlyOneRegularFile(IShellItemArray* items, std::wstring* path) {
    if (items == nullptr) {
        return false;
    }
    DWORD count = 0;
    if (FAILED(items->GetCount(&count)) || count != 1) {
        return false;
    }
    IShellItem* item = nullptr;
    if (FAILED(items->GetItemAt(0, &item)) || item == nullptr) {
        return false;
    }
    SFGAOF attributes = 0;
    const HRESULT attributeResult = item->GetAttributes(SFGAO_FOLDER | SFGAO_STREAM, &attributes);
    bool valid = SUCCEEDED(attributeResult) && (attributes & SFGAO_FOLDER) == 0 && (attributes & SFGAO_STREAM) != 0;
    if (valid && path != nullptr) {
        LPWSTR displayName = nullptr;
        valid = SUCCEEDED(item->GetDisplayName(SIGDN_FILESYSPATH, &displayName)) && displayName != nullptr;
        if (valid) {
            *path = displayName;
        }
        CoTaskMemFree(displayName);
    }
    item->Release();
    return valid;
}

HRESULT launchHelper(const std::wstring& path) {
    const std::wstring helper = siblingPath(L"upit-file-manager.exe");
    std::wstring commandLine = quoteCommandLineArgument(helper) + L" " + quoteCommandLineArgument(path);
    STARTUPINFOW startup = {};
    startup.cb = sizeof(startup);
    PROCESS_INFORMATION process = {};
    if (!CreateProcessW(nullptr, commandLine.data(), nullptr, nullptr, FALSE, CREATE_NO_WINDOW, nullptr, nullptr, &startup, &process)) {
        return HRESULT_FROM_WIN32(GetLastError());
    }
    CloseHandle(process.hThread);
    CloseHandle(process.hProcess);
    return S_OK;
}

class ExplorerCommand final : public IExplorerCommand {
public:
    ExplorerCommand() : references_(1) {
        g_serverLocks.fetch_add(1, std::memory_order_relaxed);
    }

    ~ExplorerCommand() {
        g_serverLocks.fetch_sub(1, std::memory_order_relaxed);
    }

    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID interfaceId, void** object) override {
        if (object == nullptr) {
            return E_POINTER;
        }
        *object = nullptr;
        if (interfaceId == IID_IUnknown || interfaceId == IID_IExplorerCommand) {
            *object = static_cast<IExplorerCommand*>(this);
            AddRef();
            return S_OK;
        }
        return E_NOINTERFACE;
    }

    ULONG STDMETHODCALLTYPE AddRef() override {
        return references_.fetch_add(1, std::memory_order_relaxed) + 1;
    }

    ULONG STDMETHODCALLTYPE Release() override {
        const ULONG references = references_.fetch_sub(1, std::memory_order_acq_rel) - 1;
        if (references == 0) {
            delete this;
        }
        return references;
    }

    HRESULT STDMETHODCALLTYPE GetTitle(IShellItemArray*, LPWSTR* title) override {
        return copyString(L"Upload with Upit", title);
    }

    HRESULT STDMETHODCALLTYPE GetIcon(IShellItemArray*, LPWSTR* icon) override {
        return copyString(siblingPath(L"upit-desktop.exe") + L",0", icon);
    }

    HRESULT STDMETHODCALLTYPE GetToolTip(IShellItemArray*, LPWSTR* tooltip) override {
        return copyString(L"Upload one selected regular file with Upit", tooltip);
    }

    HRESULT STDMETHODCALLTYPE GetCanonicalName(GUID* name) override {
        if (name == nullptr) {
            return E_POINTER;
        }
        *name = kUpitExplorerCommandCanonicalName;
        return S_OK;
    }

    HRESULT STDMETHODCALLTYPE GetState(IShellItemArray* items, BOOL, EXPCMDSTATE* state) override {
        if (state == nullptr) {
            return E_POINTER;
        }
        *state = isExactlyOneRegularFile(items, nullptr) ? ECS_ENABLED : ECS_HIDDEN;
        return S_OK;
    }

    HRESULT STDMETHODCALLTYPE Invoke(IShellItemArray* items, IBindCtx*) override {
        std::wstring path;
        if (!isExactlyOneRegularFile(items, &path)) {
            return E_INVALIDARG;
        }
        return launchHelper(path);
    }

    HRESULT STDMETHODCALLTYPE GetFlags(EXPCMDFLAGS* flags) override {
        if (flags == nullptr) {
            return E_POINTER;
        }
        *flags = ECF_DEFAULT;
        return S_OK;
    }

    HRESULT STDMETHODCALLTYPE EnumSubCommands(IEnumExplorerCommand** commands) override {
        if (commands == nullptr) {
            return E_POINTER;
        }
        *commands = nullptr;
        return S_FALSE;
    }

private:
    std::atomic<ULONG> references_;
};

class ExplorerCommandClassFactory final : public IClassFactory {
public:
    ExplorerCommandClassFactory() : references_(1) {
        g_serverLocks.fetch_add(1, std::memory_order_relaxed);
    }

    ~ExplorerCommandClassFactory() {
        g_serverLocks.fetch_sub(1, std::memory_order_relaxed);
    }

    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID interfaceId, void** object) override {
        if (object == nullptr) {
            return E_POINTER;
        }
        *object = nullptr;
        if (interfaceId == IID_IUnknown || interfaceId == IID_IClassFactory) {
            *object = static_cast<IClassFactory*>(this);
            AddRef();
            return S_OK;
        }
        return E_NOINTERFACE;
    }

    ULONG STDMETHODCALLTYPE AddRef() override {
        return references_.fetch_add(1, std::memory_order_relaxed) + 1;
    }

    ULONG STDMETHODCALLTYPE Release() override {
        const ULONG references = references_.fetch_sub(1, std::memory_order_acq_rel) - 1;
        if (references == 0) {
            delete this;
        }
        return references;
    }

    HRESULT STDMETHODCALLTYPE CreateInstance(IUnknown* outer, REFIID interfaceId, void** object) override {
        if (outer != nullptr) {
            return CLASS_E_NOAGGREGATION;
        }
        auto* command = new (std::nothrow) ExplorerCommand();
        if (command == nullptr) {
            return E_OUTOFMEMORY;
        }
        const HRESULT result = command->QueryInterface(interfaceId, object);
        command->Release();
        return result;
    }

    HRESULT STDMETHODCALLTYPE LockServer(BOOL lock) override {
        if (lock) {
            g_serverLocks.fetch_add(1, std::memory_order_relaxed);
        } else {
            g_serverLocks.fetch_sub(1, std::memory_order_relaxed);
        }
        return S_OK;
    }

private:
    std::atomic<ULONG> references_;
};

}  // namespace

extern "C" __declspec(dllexport) HRESULT __stdcall DllCanUnloadNow() {
    return g_serverLocks.load(std::memory_order_acquire) == 0 ? S_OK : S_FALSE;
}

extern "C" __declspec(dllexport) HRESULT __stdcall DllGetClassObject(REFCLSID classId, REFIID interfaceId, void** object) {
    if (classId != kUpitExplorerCommandClsid) {
        return CLASS_E_CLASSNOTAVAILABLE;
    }
    auto* factory = new (std::nothrow) ExplorerCommandClassFactory();
    if (factory == nullptr) {
        return E_OUTOFMEMORY;
    }
    const HRESULT result = factory->QueryInterface(interfaceId, object);
    factory->Release();
    return result;
}

BOOL APIENTRY DllMain(HMODULE module, DWORD reason, LPVOID) {
    if (reason == DLL_PROCESS_ATTACH) {
        g_module = module;
        DisableThreadLibraryCalls(module);
    }
    return TRUE;
}
