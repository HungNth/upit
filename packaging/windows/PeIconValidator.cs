using System;
using System.IO;

public static class PeIconValidator {
    public static bool HasGroupIcon(string filePath) {
        using (var fs = new FileStream(filePath, FileMode.Open, FileAccess.Read, FileShare.Read))
        using (var br = new BinaryReader(fs)) {
            if (fs.Length < 64) return false;
            if (br.ReadUInt16() != 0x5A4D) return false;
            fs.Seek(0x3C, SeekOrigin.Begin);
            uint peOffset = br.ReadUInt32();
            if (peOffset + 24 > fs.Length) return false;
            fs.Seek(peOffset, SeekOrigin.Begin);
            if (br.ReadUInt32() != 0x00004550) return false;
            ushort machine = br.ReadUInt16();
            ushort numSections = br.ReadUInt16();
            fs.Seek(12, SeekOrigin.Current);
            ushort optHeaderSize = br.ReadUInt16();
            fs.Seek(2, SeekOrigin.Current);
            
            long optHeaderOffset = fs.Position;
            ushort magic = br.ReadUInt16();
            int rvaOffset = (magic == 0x20B) ? 128 : 112;
            fs.Seek(optHeaderOffset + rvaOffset, SeekOrigin.Begin);
            uint resRva = br.ReadUInt32();
            uint resSize = br.ReadUInt32();
            if (resRva == 0 || resSize == 0) return false;
            
            fs.Seek(optHeaderOffset + optHeaderSize, SeekOrigin.Begin);
            uint rsrcRawOffset = 0;
            uint rsrcVirtualAddr = 0;
            for (int i = 0; i < numSections; i++) {
                byte[] nameBytes = br.ReadBytes(8);
                uint virtualSize = br.ReadUInt32();
                uint virtualAddr = br.ReadUInt32();
                uint rawSize = br.ReadUInt32();
                uint rawOffset = br.ReadUInt32();
                fs.Seek(16, SeekOrigin.Current);
                if (virtualAddr <= resRva && resRva < virtualAddr + virtualSize) {
                    rsrcRawOffset = rawOffset;
                    rsrcVirtualAddr = virtualAddr;
                    break;
                }
            }
            if (rsrcRawOffset == 0) return false;
            
            long rootOffset = rsrcRawOffset + (resRva - rsrcVirtualAddr);
            fs.Seek(rootOffset + 12, SeekOrigin.Begin);
            ushort namedEntries = br.ReadUInt16();
            ushort idEntries = br.ReadUInt16();
            fs.Seek(namedEntries * 8, SeekOrigin.Current);
            for (int i = 0; i < idEntries; i++) {
                uint typeId = br.ReadUInt32();
                uint offsetToData = br.ReadUInt32();
                if (typeId == 14) {
                    return true;
                }
            }
            return false;
        }
    }
}
