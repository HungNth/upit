#import "selection.h"

static NSString * const UpitServiceError = @"Upit accepts exactly one regular file.";

NSArray<NSURL *> *UpitValidatedFileURLs(NSArray<NSURL *> *urls, NSString **error) {
    if (urls.count != 1) {
        if (error != NULL) {
            *error = UpitServiceError;
        }
        return nil;
    }

    NSURL *selectedURL = urls.firstObject;
    NSNumber *isDirectory = nil;
    NSNumber *isRegularFile = nil;
    if (!selectedURL.isFileURL ||
        ![selectedURL getResourceValue:&isDirectory forKey:NSURLIsDirectoryKey error:nil] ||
        ![selectedURL getResourceValue:&isRegularFile forKey:NSURLIsRegularFileKey error:nil] ||
        isDirectory.boolValue || !isRegularFile.boolValue) {
        if (error != NULL) {
            *error = UpitServiceError;
        }
        return nil;
    }
    return @[selectedURL];
}
