package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const UpitAUMID = "HungNth.Upit"

func ensureStartMenuShortcut(desktopExePath string) error {
	programsDir := os.Getenv("APPDATA")
	if programsDir == "" {
		return fmt.Errorf("APPDATA environment variable is empty")
	}
	targetDir := filepath.Join(programsDir, "Microsoft", "Windows", "Start Menu", "Programs")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return err
	}
	shortcutPath := filepath.Join(targetDir, "Upit.lnk")
	return createShortcutWithAUMID(shortcutPath, desktopExePath, UpitAUMID)
}

func removeStartMenuShortcut() error {
	programsDir := os.Getenv("APPDATA")
	if programsDir == "" {
		return nil
	}
	shortcutPath := filepath.Join(programsDir, "Microsoft", "Windows", "Start Menu", "Programs", "Upit.lnk")
	if err := os.Remove(shortcutPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func createShortcutWithAUMID(shortcutPath, targetExePath, aumid string) error {
	psScript := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$shortcutPath = '%s'
$targetPath = '%s'
$aumid = '%s'

Add-Type -TypeDefinition @'
using System;
using System.IO;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.ComTypes;
using System.Text;

public class Win32Lnk : IDisposable {
    [ComImport, Guid("000214F9-0000-0000-C000-000000000046"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    private interface IShellLinkW {
        uint GetPath([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszFile, int cch, IntPtr pfd, uint fFlags);
        uint GetIDList(out IntPtr ppidl);
        uint SetIDList(IntPtr pidl);
        uint GetDescription([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszName, int cch);
        uint SetDescription([MarshalAs(UnmanagedType.LPWStr)] string pszName);
        uint GetWorkingDirectory([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszDir, int cch);
        uint SetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] string pszDir);
        uint GetArguments([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszArgs, int cch);
        uint SetArguments([MarshalAs(UnmanagedType.LPWStr)] string pszArgs);
        uint GetHotKey(out ushort pwHotkey);
        uint SetHotKey(ushort wHotkey);
        uint GetShowCmd(out int piShowCmd);
        uint SetShowCmd(int iShowCmd);
        uint GetIconLocation([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszIconPath, int cch, out int piIcon);
        uint SetIconLocation([MarshalAs(UnmanagedType.LPWStr)] string pszIconPath, int iIcon);
        uint SetRelativePath([MarshalAs(UnmanagedType.LPWStr)] string pszPathRel, uint dwReserved);
        uint Resolve(IntPtr hwnd, uint fFlags);
        uint SetPath([MarshalAs(UnmanagedType.LPWStr)] string pszFile);
    }

    [ComImport, ClassInterface(ClassInterfaceType.None), Guid("00021401-0000-0000-C000-000000000046")]
    private class CShellLink {}

    [ComImport, InterfaceType(ComInterfaceType.InterfaceIsIUnknown), Guid("0000010b-0000-0000-C000-000000000046")]
    private interface IPersistFile {
        uint GetClassID(out Guid pClassID);
        uint IsDirty();
        uint Load([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, uint dwMode);
        uint Save([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, bool fRemember);
        uint SaveCompleted([MarshalAs(UnmanagedType.LPWStr)] string pszFileName);
        uint GetCurFile([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszFile);
    }

    [ComImport, InterfaceType(ComInterfaceType.InterfaceIsIUnknown), Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99")]
    private interface IPropertyStore {
        uint GetCount([Out] out uint cProps);
        uint GetAt([In] uint iProp, out PropertyKey pkey);
        uint GetValue([In] ref PropertyKey key, [Out] PropVariant pv);
        uint SetValue([In] ref PropertyKey key, [In] PropVariant pv);
        uint Commit();
    }

    [StructLayout(LayoutKind.Sequential, Pack = 4)]
    private struct PropertyKey {
        public Guid formatId;
        public Int32 propertyId;
        public PropertyKey(Guid guid, Int32 pid) {
            formatId = guid;
            propertyId = pid;
        }
    }

    [StructLayout(LayoutKind.Explicit)]
    private sealed class PropVariant : IDisposable {
        [FieldOffset(0)] ushort valueType;
        [FieldOffset(8)] IntPtr ptr;

        public PropVariant() {}
        public PropVariant(string value) {
            valueType = 31; // VT_LPWSTR
            ptr = Marshal.StringToCoTaskMemUni(value);
        }
        public string Value {
            get { return Marshal.PtrToStringUni(ptr); }
        }
        ~PropVariant() { Dispose(); }
        public void Dispose() {
            PropVariantClear(this);
            GC.SuppressFinalize(this);
        }
    }

    [DllImport("Ole32.dll", PreserveSig = false)]
    private static extern void PropVariantClear([In, Out] PropVariant pvar);

    private static readonly PropertyKey AppUserModelIDKey = new PropertyKey(new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"), 5);

    public static void Create(string shortcutPath, string targetPath, string aumid) {
        object linkObj = new CShellLink();
        IShellLinkW link = (IShellLinkW)linkObj;
        uint hr = link.SetPath(targetPath);
        if (hr > 1) throw new COMException("SetPath failed", (int)hr);

        IPropertyStore store = (IPropertyStore)linkObj;
        using (PropVariant pv = new PropVariant(aumid)) {
            PropertyKey key = AppUserModelIDKey;
            hr = store.SetValue(ref key, pv);
            if (hr > 1) throw new COMException("SetValue failed", (int)hr);
            hr = store.Commit();
            if (hr > 1) throw new COMException("Commit failed", (int)hr);
        }

        IPersistFile file = (IPersistFile)linkObj;
        hr = file.Save(shortcutPath, true);
        if (hr > 1) throw new COMException("Save failed", (int)hr);
    }

    public static string Read(string shortcutPath) {
        object linkObj = new CShellLink();
        IPersistFile file = (IPersistFile)linkObj;
        uint hr = file.Load(shortcutPath, 0);
        if (hr > 1) throw new COMException("Load failed", (int)hr);

        IPropertyStore store = (IPropertyStore)linkObj;
        using (PropVariant pv = new PropVariant()) {
            PropertyKey key = AppUserModelIDKey;
            hr = store.GetValue(ref key, pv);
            if (hr > 1) throw new COMException("GetValue failed", (int)hr);
            return pv.Value;
        }
    }

    public void Dispose() {}
}
'@

[Win32Lnk]::Create($shortcutPath, $targetPath, $aumid)
`, strings.ReplaceAll(shortcutPath, `'`, `''`), strings.ReplaceAll(targetExePath, `'`, `''`), strings.ReplaceAll(aumid, `'`, `''`))

	cmd := exec.CommandContext(context.Background(), "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create shortcut with AUMID failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func readShortcutAUMID(shortcutPath string) (string, error) {
	psScript := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$shortcutPath = '%s'

Add-Type -TypeDefinition @'
using System;
using System.IO;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.ComTypes;
using System.Text;

public class Win32LnkReader : IDisposable {
    [ComImport, Guid("000214F9-0000-0000-C000-000000000046"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    private interface IShellLinkW {
        uint GetPath([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszFile, int cch, IntPtr pfd, uint fFlags);
        uint GetIDList(out IntPtr ppidl);
        uint SetIDList(IntPtr pidl);
        uint GetDescription([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszName, int cch);
        uint SetDescription([MarshalAs(UnmanagedType.LPWStr)] string pszName);
        uint GetWorkingDirectory([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszDir, int cch);
        uint SetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] string pszDir);
        uint GetArguments([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszArgs, int cch);
        uint SetArguments([MarshalAs(UnmanagedType.LPWStr)] string pszArgs);
        uint GetHotKey(out ushort pwHotkey);
        uint SetHotKey(ushort wHotkey);
        uint GetShowCmd(out int piShowCmd);
        uint SetShowCmd(int iShowCmd);
        uint GetIconLocation([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszIconPath, int cch, out int piIcon);
        uint SetIconLocation([MarshalAs(UnmanagedType.LPWStr)] string pszIconPath, int iIcon);
        uint SetRelativePath([MarshalAs(UnmanagedType.LPWStr)] string pszPathRel, uint dwReserved);
        uint Resolve(IntPtr hwnd, uint fFlags);
        uint SetPath([MarshalAs(UnmanagedType.LPWStr)] string pszFile);
    }

    [ComImport, ClassInterface(ClassInterfaceType.None), Guid("00021401-0000-0000-C000-000000000046")]
    private class CShellLink {}

    [ComImport, InterfaceType(ComInterfaceType.InterfaceIsIUnknown), Guid("0000010b-0000-0000-C000-000000000046")]
    private interface IPersistFile {
        uint GetClassID(out Guid pClassID);
        uint IsDirty();
        uint Load([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, uint dwMode);
        uint Save([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, bool fRemember);
        uint SaveCompleted([MarshalAs(UnmanagedType.LPWStr)] string pszFileName);
        uint GetCurFile([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszFile);
    }

    [ComImport, InterfaceType(ComInterfaceType.InterfaceIsIUnknown), Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99")]
    private interface IPropertyStore {
        uint GetCount([Out] out uint cProps);
        uint GetAt([In] uint iProp, out PropertyKey pkey);
        uint GetValue([In] ref PropertyKey key, [Out] PropVariant pv);
        uint SetValue([In] ref PropertyKey key, [In] PropVariant pv);
        uint Commit();
    }

    [StructLayout(LayoutKind.Sequential, Pack = 4)]
    private struct PropertyKey {
        public Guid formatId;
        public Int32 propertyId;
        public PropertyKey(Guid guid, Int32 pid) {
            formatId = guid;
            propertyId = pid;
        }
    }

    [StructLayout(LayoutKind.Explicit)]
    private sealed class PropVariant : IDisposable {
        [FieldOffset(0)] ushort valueType;
        [FieldOffset(8)] IntPtr ptr;

        public PropVariant() {}
        public string Value {
            get { return Marshal.PtrToStringUni(ptr); }
        }
        ~PropVariant() { Dispose(); }
        public void Dispose() {
            PropVariantClear(this);
            GC.SuppressFinalize(this);
        }
    }

    [DllImport("Ole32.dll", PreserveSig = false)]
    private static extern void PropVariantClear([In, Out] PropVariant pvar);

    private static readonly PropertyKey AppUserModelIDKey = new PropertyKey(new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"), 5);

    public static string Read(string shortcutPath) {
        object linkObj = new CShellLink();
        IPersistFile file = (IPersistFile)linkObj;
        uint hr = file.Load(shortcutPath, 0);
        if (hr > 1) throw new COMException("Load failed", (int)hr);

        IPropertyStore store = (IPropertyStore)linkObj;
        using (PropVariant pv = new PropVariant()) {
            PropertyKey key = AppUserModelIDKey;
            hr = store.GetValue(ref key, pv);
            if (hr > 1) throw new COMException("GetValue failed", (int)hr);
            return pv.Value;
        }
    }

    public void Dispose() {}
}
'@

[Win32LnkReader]::Read($shortcutPath)
`, strings.ReplaceAll(shortcutPath, `'`, `''`))

	cmd := exec.CommandContext(context.Background(), "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read shortcut AUMID failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}
