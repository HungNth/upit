//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <CoreServices/CoreServices.h>
#include <stdbool.h>

bool upitIntegrationRegistered(const char *identifier, const char *path) {
    @autoreleasepool {
        NSString *bundleID = [NSString stringWithUTF8String:identifier];
        NSString *expectedPath = [NSString stringWithUTF8String:path];
        CFArrayRef applications = LSCopyApplicationURLsForBundleIdentifier((__bridge CFStringRef)bundleID, NULL);
        if (!applications) return false;
        bool found = false;
        for (NSURL *url in (__bridge NSArray *)applications) {
            if ([[url.path stringByResolvingSymlinksInPath] isEqualToString:[expectedPath stringByResolvingSymlinksInPath]]) { found = true; break; }
        }
        CFRelease(applications);
        return found;
    }
}
void upitIntegrationRefresh(void) { @autoreleasepool { NSUpdateDynamicServices(); } }
