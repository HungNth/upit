using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

public static class ConsoleCancellationSmoke {
    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct StartupInfo {
        public int size;
        public string reserved, desktop, title;
        public int x, y, xSize, ySize, xChars, yChars, fill, flags;
        public short show, reservedSize;
        public IntPtr reservedBytes, stdin, stdout, stderr;
    }
    [StructLayout(LayoutKind.Sequential)]
    private struct ProcessInfo { public IntPtr process, thread; public int processId, threadId; }
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern bool CreateProcess(string application, StringBuilder command, IntPtr processAttributes, IntPtr threadAttributes, bool inherit, uint flags, IntPtr environment, string directory, ref StartupInfo startup, out ProcessInfo process);
    [DllImport("kernel32.dll", SetLastError = true)] private static extern bool AttachConsole(uint processId);
    [DllImport("kernel32.dll")] private static extern bool FreeConsole();
    [DllImport("kernel32.dll", SetLastError = true)] private static extern bool GenerateConsoleCtrlEvent(uint control, uint processGroup);
    [DllImport("kernel32.dll", SetLastError = true)] private static extern uint WaitForSingleObject(IntPtr handle, uint milliseconds);
    [DllImport("kernel32.dll", SetLastError = true)] private static extern bool GetExitCodeProcess(IntPtr handle, out uint code);
    [DllImport("kernel32.dll")] private static extern bool CloseHandle(IntPtr handle);

    public static int StartUpload(string launcher, string file, string directory) {
        var startup = new StartupInfo();
        startup.size = Marshal.SizeOf(typeof(StartupInfo));
        startup.flags = 1; // STARTF_USESHOWWINDOW: hide the test console.
        startup.show = 0;
        ProcessInfo process;
        var command = new StringBuilder("\"" + launcher + "\" upload --timeout 5m \"" + file + "\"");
        if (!CreateProcess(launcher, command, IntPtr.Zero, IntPtr.Zero, false, 0x210, IntPtr.Zero, directory, ref startup, out process))
            throw new Win32Exception(Marshal.GetLastWin32Error());
        CloseHandle(process.thread);
        CloseHandle(process.process);
        return process.processId;
    }

    public static int CancelProcess(int processId) {
        using (var process = System.Diagnostics.Process.GetProcessById(processId)) {
            if (process.HasExited) throw new InvalidOperationException("Upload finished before cancellation.");
            IntPtr handle = process.Handle;
            FreeConsole();
            if (!AttachConsole((uint)processId)) throw new Win32Exception(Marshal.GetLastWin32Error());
            try {
                if (!GenerateConsoleCtrlEvent(1, (uint)processId)) throw new Win32Exception(Marshal.GetLastWin32Error());
                if (WaitForSingleObject(handle, 10000) != 0) throw new InvalidOperationException("Launcher did not exit after console cancellation.");
                uint exit;
                if (!GetExitCodeProcess(handle, out exit)) throw new Win32Exception(Marshal.GetLastWin32Error());
                if (exit == 0) throw new InvalidOperationException("Cancelled upload reported success.");
                return unchecked((int)exit);
            } finally { FreeConsole(); }
        }
    }

    public static int CancelUpload(string launcher, string file, string directory) {
        int processId = StartUpload(launcher, file, directory);
        System.Threading.Thread.Sleep(1000);
        return CancelProcess(processId);
    }
}
