#import "selection.h"

static NSString * const UpitServiceError = @"Upit accepts exactly one regular file.";

NSArray<NSURL *> *UpitValidatedFileURLs(NSArray<NSURL *> *urls, NSString **error) {
    if (urls.count != 1) {
        if (error != NULL) {
            *error = UpitServiceError;
        }
        return nil;
    }

    // Finder sends file reference URLs (file:///.file/id=...); resolve to a path URL for the helper.
    NSURL *selectedURL = urls.firstObject.filePathURL;
    NSNumber *isDirectory = nil;
    NSNumber *isRegularFile = nil;
    if (selectedURL == nil ||
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
