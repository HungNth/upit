#import <Cocoa/Cocoa.h>
#import "selection.h"

@interface UpitServiceProvider : NSObject <NSApplicationDelegate>
@end

@implementation UpitServiceProvider

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    [NSApp setServicesProvider:self];
}

- (void)uploadFileService:(NSPasteboard *)pasteboard
                 userData:(NSString *)userData
                    error:(NSString **)error {
    NSArray<NSURL *> *urls = [pasteboard readObjectsForClasses:@[[NSURL class]]
                                                        options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
    NSString *selectionError = nil;
    NSArray<NSURL *> *validatedURLs = UpitValidatedFileURLs(urls, &selectionError);
    if (validatedURLs == nil) {
        if (error != NULL) {
            *error = selectionError;
        }
        return;
    }

    NSURL *selectedURL = validatedURLs.firstObject;

    NSString *helperPath = [[NSBundle mainBundle].bundlePath stringByAppendingPathComponent:
        @"Contents/Helpers/UpitFileManager.app/Contents/MacOS/upit-file-manager"];
    if (![[NSFileManager defaultManager] isExecutableFileAtPath:helperPath]) {
        if (error != NULL) {
            *error = @"Upit upload is unavailable.";
        }
        return;
    }

    NSTask *task = [NSTask new];
    task.executableURL = [NSURL fileURLWithPath:helperPath];
    task.arguments = @[@"--file-url", selectedURL.absoluteString ?: @""];
    task.standardOutput = [NSFileHandle fileHandleWithNullDevice];
    task.standardError = [NSFileHandle fileHandleWithNullDevice];

    NSError *launchError = nil;
    if (![task launchAndReturnError:&launchError] && error != NULL) {
        *error = @"Upit upload could not start.";
    }

    [NSApp terminate:nil];
}

@end

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        NSApplication *application = [NSApplication sharedApplication];
        [application setActivationPolicy:NSApplicationActivationPolicyAccessory];

        UpitServiceProvider *provider = [UpitServiceProvider new];
        [application setDelegate:provider];
        [application run];
    }
    return 0;
}
