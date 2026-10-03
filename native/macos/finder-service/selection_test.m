#import "selection.h"

static int fail(NSString *message) {
    NSLog(@"selection test failed: %@", message);
    return 1;
}

int main(void) {
    @autoreleasepool {
        NSString *directory = [NSTemporaryDirectory() stringByAppendingPathComponent:
            [NSString stringWithFormat:@"upit-finder-selection-%@", NSUUID.UUID.UUIDString]];
        NSString *filePath = [directory stringByAppendingPathComponent:@"payload.txt"];
        NSFileManager *fileManager = [NSFileManager defaultManager];
        if (![fileManager createDirectoryAtPath:directory withIntermediateDirectories:YES attributes:nil error:nil]) {
            return fail(@"could not create fixture directory");
        }
        NSData *contents = [@"payload" dataUsingEncoding:NSUTF8StringEncoding];
        if (![fileManager createFileAtPath:filePath contents:contents attributes:nil]) {
            return fail(@"could not create regular-file fixture");
        }

        NSURL *fileURL = [NSURL fileURLWithPath:filePath];
        NSString *error = nil;
        NSArray<NSURL *> *selected = UpitValidatedFileURLs(@[fileURL], &error);
        if (selected.count != 1 || ![selected.firstObject isEqual:fileURL]) {
            return fail(@"one regular file was rejected");
        }
        if (UpitValidatedFileURLs(@[], &error) != nil) {
            return fail(@"empty selection was accepted");
        }
        if (UpitValidatedFileURLs(@[fileURL, fileURL], &error) != nil) {
            return fail(@"multiple selection was accepted");
        }
        NSURL *referenceURL = [fileURL fileReferenceURL];
        selected = UpitValidatedFileURLs(@[referenceURL], &error);
        if (selected.count != 1 || selected.firstObject.isFileReferenceURL ||
            ![selected.firstObject.URLByResolvingSymlinksInPath.path
                isEqualToString:fileURL.URLByResolvingSymlinksInPath.path]) {
            return fail(@"Finder file reference URL was not converted to a path URL");
        }

        NSURL *directoryURL = [NSURL fileURLWithPath:directory isDirectory:YES];
        if (UpitValidatedFileURLs(@[directoryURL], &error) != nil) {
            return fail(@"directory selection was accepted");
        }
        NSURL *remoteURL = [NSURL URLWithString:@"https://example.test/payload.txt"];
        if (UpitValidatedFileURLs(@[remoteURL], &error) != nil) {
            return fail(@"non-file URL was accepted");
        }

        [fileManager removeItemAtPath:directory error:nil];
    }
    return 0;
}
