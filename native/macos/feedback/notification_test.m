#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>
#import <objc/runtime.h>
#include <stdint.h>

void upitFeedbackBegin(void);
void upitFeedbackProgress(const char *, int64_t, int64_t);
void upitFeedbackComplete(const char *, int, int, int);
void upitFeedbackClose(void);
int upitFeedbackNotify(const char *, const char *, const char *);
void upitFeedbackPrepareEventLoop(void);
int upitFeedbackAwaitLaunchContext(void);

@interface TestNotificationCenter : NSObject
@property BOOL authorized;
@property BOOL failDelivery;
@property BOOL modalShown;
@property(nonatomic, strong) NSMutableArray<UNNotificationRequest *> *requests;
@end

static TestNotificationCenter *testCenter;

@implementation TestNotificationCenter
- (void)requestAuthorizationWithOptions:(UNAuthorizationOptions)options completionHandler:(void (^)(BOOL, NSError *))completion {
    completion(self.authorized, nil);
}
- (void)setNotificationCategories:(NSSet *)categories {}
- (void)setDelegate:(id)delegate {}
- (void)addNotificationRequest:(UNNotificationRequest *)request withCompletionHandler:(void (^)(NSError *))completion {
    if (!self.failDelivery) { [self.requests addObject:request]; }
    if (completion != nil) {
        completion(self.failDelivery ? [NSError errorWithDomain:@"test" code:1 userInfo:nil] : nil);
    }
}
- (void)removePendingNotificationRequestsWithIdentifiers:(NSArray<NSString *> *)identifiers {
    NSIndexSet *indices = [self.requests indexesOfObjectsPassingTest:^BOOL(UNNotificationRequest *request, NSUInteger index, BOOL *stop) {
        return [identifiers containsObject:request.identifier];
    }];
    [self.requests removeObjectsAtIndexes:indices];
}
- (void)removeDeliveredNotificationsWithIdentifiers:(NSArray<NSString *> *)identifiers {
    [self removePendingNotificationRequestsWithIdentifiers:identifiers];
}
@end

static id currentTestCenter(id object, SEL selector) { return testCenter; }
static NSModalResponse recordModal(id object, SEL selector) {
    testCenter.modalShown = YES;
    return NSAlertFirstButtonReturn;
}
static void require(BOOL condition, NSString *message) {
    if (!condition) { NSLog(@"FAIL: %@", message); exit(1); }
}

int main(void) {
    @autoreleasepool {
        testCenter = [TestNotificationCenter new];
        testCenter.requests = [NSMutableArray array];
        method_setImplementation(class_getClassMethod(UNUserNotificationCenter.class, @selector(currentNotificationCenter)), (IMP)currentTestCenter);
        method_setImplementation(class_getInstanceMethod(NSAlert.class, @selector(runModal)), (IMP)recordModal);

        upitFeedbackPrepareEventLoop();
        [[NSNotificationCenter defaultCenter] postNotificationName:NSApplicationDidFinishLaunchingNotification
            object:NSApp userInfo:@{}];
        require(upitFeedbackAwaitLaunchContext() == 0, @"ordinary app launch was treated as notification activation");
        upitFeedbackPrepareEventLoop();
        [[NSNotificationCenter defaultCenter] postNotificationName:NSApplicationDidFinishLaunchingNotification
            object:NSApp userInfo:@{NSApplicationLaunchUserNotificationKey: @YES}];
        require(upitFeedbackAwaitLaunchContext() == 1, @"Notification Center launch context was not recognized");

        testCenter.authorized = YES;
        upitFeedbackBegin();
        require(testCenter.requests.lastObject.content.sound == nil, @"initial progress is audible");
        upitFeedbackProgress("uploading", 10, 100);
        require(testCenter.requests.lastObject.content.sound == nil, @"updated progress is audible");
        upitFeedbackComplete("Upload completed with a warning.", 1, 0, 0);
        require(testCenter.requests.lastObject.content.sound != nil, @"actionable terminal notification lost its existing sound");
        upitFeedbackClose();
        [testCenter.requests removeAllObjects];
        upitFeedbackBegin();
        upitFeedbackClose();
        require(testCenter.requests.count == 0, @"progress remains after Close");
        require(upitFeedbackNotify("first-event-00001", "Upload complete", "Final URL copied to clipboard.") == 0, @"authorized notification failed");
        upitFeedbackClose();
        require(testCenter.requests.count == 1, @"deferred Close removed terminal notification");
        UNNotificationRequest *first = testCenter.requests.firstObject;
        require([first.content.title isEqual:@"Upload complete"] && [first.content.body isEqual:@"Final URL copied to clipboard."], @"incorrect native content");
        require(first.content.sound == nil && first.content.categoryIdentifier.length == 0 && first.content.userInfo.count == 0 && first.trigger == nil, @"terminal request has sound, actions, private payload, or delay");
        require(upitFeedbackNotify("second-event-002", "Upload complete", "Final URL copied to clipboard.") == 0, @"second notification failed");
        require(testCenter.requests.count == 2 && ![first.identifier isEqual:testCenter.requests.lastObject.identifier], @"terminal events replace one another");

        testCenter.failDelivery = YES;
        require(upitFeedbackNotify("failed-event-003", "Upload complete", "Final URL copied to clipboard.") == 2, @"API failure was not reported");
        testCenter.failDelivery = NO;
        testCenter.authorized = NO;
        upitFeedbackBegin();
        upitFeedbackClose();
        require(upitFeedbackNotify("denied-event-004", "Upload complete", "Final URL copied to clipboard.") == 1, @"denied delivery was not reported");
        require(testCenter.requests.count == 2 && !testCenter.modalShown, @"failure/denial posted a request or opened modal fallback");
        puts("macOS native clean-success notification tests passed.");
    }
    return 0;
}
