#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>
#include <stdatomic.h>
#include <dispatch/dispatch.h>
#include <stdint.h>

static const int UPIT_ACTION_CANCEL = 1;
static const int UPIT_ACTION_COPY = 2;
static const int UPIT_ACTION_RETRY = 3;
static const int UPIT_ACTION_OPEN = 4;
static const int UPIT_ACTION_DISMISS = -1;

static _Atomic int upitAction = 0;
static _Atomic int upitNotificationsAuthorized = 0;
static atomic_bool upitStopEventLoop = false;
static NSUInteger upitNotificationSequence = 0;

@interface UpitNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation UpitNotificationDelegate

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       didReceiveNotificationResponse:(UNNotificationResponse *)response
                withCompletionHandler:(void (^)(void))completionHandler {
    NSString *identifier = response.actionIdentifier;
    int action = 0;
    if ([identifier isEqualToString:@"com.hungnth.upit.cancel"]) {
        action = UPIT_ACTION_CANCEL;
    } else if ([identifier isEqualToString:@"com.hungnth.upit.copy"]) {
        action = UPIT_ACTION_COPY;
    } else if ([identifier isEqualToString:@"com.hungnth.upit.retry"]) {
        action = UPIT_ACTION_RETRY;
    } else if ([identifier isEqualToString:@"com.hungnth.upit.open-desktop"]) {
        action = UPIT_ACTION_OPEN;
    } else if ([identifier isEqualToString:UNNotificationDismissActionIdentifier]) {
        action = UPIT_ACTION_DISMISS;
    }
    if (action != 0) {
        atomic_store(&upitAction, action);
    }
    completionHandler();
}

@end

@interface UpitProgressUpdate : NSObject
@property(nonatomic, copy) NSString *body;
@property(nonatomic) int64_t processed;
@property(nonatomic) int64_t total;
@end

@implementation UpitProgressUpdate
@end

@interface UpitFallbackController : NSObject <NSWindowDelegate>
@property(nonatomic, strong) NSPanel *panel;
@property(nonatomic, strong) NSTextField *label;
@property(nonatomic, strong) NSProgressIndicator *progress;
@property(nonatomic) BOOL suppressCancel;
- (void)beginOnMainThread:(id)object;
- (void)updateOnMainThread:(UpitProgressUpdate *)update;
- (void)closeOnMainThread:(id)object;
- (void)cancel:(id)sender;
@end

@implementation UpitFallbackController

- (void)beginOnMainThread:(id)object {
    if (self.panel != nil) {
        return;
    }
    NSRect frame = NSMakeRect(0, 0, 460, 150);
    self.panel = [[NSPanel alloc] initWithContentRect:frame
                                             styleMask:(NSWindowStyleMaskTitled | NSWindowStyleMaskClosable)
                                               backing:NSBackingStoreBuffered
                                                 defer:NO];
    self.panel.title = @"Upit";
    self.panel.level = NSFloatingWindowLevel;
    self.panel.floatingPanel = YES;
    self.panel.hidesOnDeactivate = NO;
    self.panel.delegate = self;

    self.label = [[NSTextField alloc] initWithFrame:NSMakeRect(20, 92, 420, 28)];
    self.label.editable = NO;
    self.label.bordered = NO;
    self.label.drawsBackground = NO;
    self.label.stringValue = @"Upload in progress. Select Cancel to stop.";
    [self.panel.contentView addSubview:self.label];

    self.progress = [[NSProgressIndicator alloc] initWithFrame:NSMakeRect(20, 62, 420, 18)];
    self.progress.style = NSProgressIndicatorStyleBar;
    self.progress.indeterminate = YES;
    [self.progress startAnimation:nil];
    [self.panel.contentView addSubview:self.progress];

    NSButton *cancel = [[NSButton alloc] initWithFrame:NSMakeRect(350, 18, 90, 28)];
    cancel.title = @"Cancel";
    cancel.bezelStyle = NSBezelStyleRounded;
    cancel.target = self;
    cancel.action = @selector(cancel:);
    [self.panel.contentView addSubview:cancel];

    [self.panel center];
    [self.panel makeKeyAndOrderFront:nil];
}

- (void)updateOnMainThread:(UpitProgressUpdate *)update {
    if (self.panel == nil) {
        return;
    }
    self.label.stringValue = update.body;
    if (update.total > 0) {
        self.progress.indeterminate = NO;
        self.progress.maxValue = (double)update.total;
        self.progress.doubleValue = (double)update.processed;
    } else {
        self.progress.indeterminate = YES;
        [self.progress startAnimation:nil];
    }
}

- (void)cancel:(id)sender {
    atomic_store(&upitAction, UPIT_ACTION_CANCEL);
    [self closeOnMainThread:nil];
}

- (BOOL)windowShouldClose:(NSWindow *)sender {
    if (!self.suppressCancel) {
        atomic_store(&upitAction, UPIT_ACTION_CANCEL);
    }
    return YES;
}

- (void)closeOnMainThread:(id)object {
    self.suppressCancel = YES;
    [self.progress stopAnimation:nil];
    [self.panel orderOut:nil];
    self.panel = nil;
    self.label = nil;
    self.progress = nil;
}

@end

@interface UpitAlertRequest : NSObject
@property(nonatomic, copy) NSString *message;
@property(nonatomic) BOOL copyAction;
@property(nonatomic) BOOL retryAction;
@property(nonatomic) BOOL openDesktopAction;
@property(nonatomic) int result;
- (void)runOnMainThread:(id)object;
@end

@implementation UpitAlertRequest

- (void)runOnMainThread:(id)object {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    NSAlert *alert = [NSAlert new];
    alert.messageText = @"Upit";
    alert.informativeText = self.message ?: @"";
    alert.alertStyle = NSAlertStyleInformational;
    BOOL hasActions = self.copyAction || self.retryAction || self.openDesktopAction;
    if (self.copyAction) {
        [alert addButtonWithTitle:@"Copy Final URL"];
    }
    if (self.retryAction) {
        [alert addButtonWithTitle:@"Retry"];
    }
    if (self.openDesktopAction) {
        [alert addButtonWithTitle:@"Open Upit Desktop"];
    }
    if (!hasActions) {
        [alert addButtonWithTitle:@"OK"];
    }

    NSModalResponse response = [alert runModal];
    if (!hasActions) {
        self.result = 0;
        return;
    }
    NSUInteger index = (NSUInteger)(response - NSAlertFirstButtonReturn);
    NSUInteger actionIndex = 0;
    if (self.copyAction && index == actionIndex++) {
        self.result = UPIT_ACTION_COPY;
        return;
    }
    if (self.retryAction && index == actionIndex++) {
        self.result = UPIT_ACTION_RETRY;
        return;
    }
    if (self.openDesktopAction && index == actionIndex) {
        self.result = UPIT_ACTION_OPEN;
    }
}

@end

static UpitNotificationDelegate *upitDelegate;
static UpitFallbackController *upitFallback;
static dispatch_semaphore_t upitPermissionFinished;

static NSString *upitString(const char *value) {
    if (value == NULL) {
        return @"";
    }
    NSString *string = [NSString stringWithUTF8String:value];
    return string ?: @"";
}

static void upitRunOnMainThread(id target, SEL selector, id object, BOOL wait) {
    if ([NSThread isMainThread]) {
        [target performSelector:selector withObject:object];
        return;
    }
    [target performSelectorOnMainThread:selector withObject:object waitUntilDone:wait];
}
@interface UpitNotificationSetup : NSObject
- (void)configureOnMainThread:(id)object;
@end

@implementation UpitNotificationSetup

- (void)configureOnMainThread:(id)object {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

    UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
    if (upitDelegate == nil) {
        upitDelegate = [UpitNotificationDelegate new];
    }
    center.delegate = upitDelegate;
    upitPermissionFinished = dispatch_semaphore_create(0);
    [center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
                          completionHandler:^(BOOL granted, NSError *error) {
        atomic_store(&upitNotificationsAuthorized, granted ? 1 : 0);
        dispatch_semaphore_signal(upitPermissionFinished);
    }];
}

@end


static void upitConfigureNotifications(void) {
    UpitNotificationSetup *setup = [UpitNotificationSetup new];
    upitRunOnMainThread(setup, @selector(configureOnMainThread:), nil, YES);
    dispatch_semaphore_t finished = upitPermissionFinished;
    if (finished != nil) {
        dispatch_semaphore_wait(finished, dispatch_time(DISPATCH_TIME_NOW, (int64_t)(500 * NSEC_PER_MSEC)));
    }
}

static void upitRegisterActiveCategory(void) {
    UNNotificationAction *cancel = [UNNotificationAction actionWithIdentifier:@"com.hungnth.upit.cancel"
                                                                          title:@"Cancel"
                                                                        options:UNNotificationActionOptionDestructive];
    UNNotificationCategory *category = [UNNotificationCategory categoryWithIdentifier:@"com.hungnth.upit.active"
                                                                                  actions:@[cancel]
                                                                        intentIdentifiers:@[]
                                                                                  options:0];
    [[UNUserNotificationCenter currentNotificationCenter] setNotificationCategories:[NSSet setWithObject:category]];
}

static void upitRegisterTerminalCategory(BOOL copy, BOOL retry, BOOL openDesktop) {
    NSMutableArray<UNNotificationAction *> *actions = [NSMutableArray array];
    if (copy) {
        [actions addObject:[UNNotificationAction actionWithIdentifier:@"com.hungnth.upit.copy"
                                                                 title:@"Copy Final URL"
                                                               options:0]];
    }
    if (retry) {
        [actions addObject:[UNNotificationAction actionWithIdentifier:@"com.hungnth.upit.retry"
                                                                 title:@"Retry"
                                                               options:0]];
    }
    if (openDesktop) {
        [actions addObject:[UNNotificationAction actionWithIdentifier:@"com.hungnth.upit.open-desktop"
                                                                 title:@"Open Upit Desktop"
                                                               options:0]];
    }
    UNNotificationCategory *category = [UNNotificationCategory categoryWithIdentifier:@"com.hungnth.upit.terminal"
                                                                                  actions:actions
                                                                        intentIdentifiers:@[]
                                                                                  options:0];
    [[UNUserNotificationCenter currentNotificationCenter] setNotificationCategories:[NSSet setWithObject:category]];
}

static void upitPostNotification(NSString *title, NSString *body, NSString *category) {
    if (atomic_load(&upitNotificationsAuthorized) != 1) {
        return;
    }
    UNMutableNotificationContent *content = [UNMutableNotificationContent new];
    content.title = title;
    content.body = body;
    content.categoryIdentifier = category;
    content.sound = [UNNotificationSound defaultSound];

    NSString *identifier = [NSString stringWithFormat:@"com.hungnth.upit.file-manager.%lu", (unsigned long)++upitNotificationSequence];
    UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:identifier
                                                                            content:content
                                                                            trigger:nil];
    [[UNUserNotificationCenter currentNotificationCenter] addNotificationRequest:request withCompletionHandler:nil];
}

static void upitStartFallback(void) {
    if (upitFallback != nil) {
        return;
    }
    upitFallback = [UpitFallbackController new];
    upitRunOnMainThread(upitFallback, @selector(beginOnMainThread:), nil, YES);
}

static void upitUpdateFallback(NSString *body, int64_t processed, int64_t total) {
    if (upitFallback == nil) {
        return;
    }
    UpitProgressUpdate *update = [UpitProgressUpdate new];
    update.body = body;
    update.processed = processed;
    update.total = total;
    upitRunOnMainThread(upitFallback, @selector(updateOnMainThread:), update, NO);
}

static void upitStopFallback(void) {
    if (upitFallback == nil) {
        return;
    }
    UpitFallbackController *fallback = upitFallback;
    upitRunOnMainThread(fallback, @selector(closeOnMainThread:), nil, YES);
    upitFallback = nil;
}

void upitFeedbackBegin(void) {
    @autoreleasepool {
        atomic_store(&upitAction, 0);
        upitConfigureNotifications();
        upitRegisterActiveCategory();
        if (atomic_load(&upitNotificationsAuthorized) == 1) {
            upitPostNotification(@"Upit", @"Preparing upload…", @"com.hungnth.upit.active");
        } else {
            upitStartFallback();
        }
    }
}

void upitFeedbackProgress(const char *phase, int64_t processed, int64_t total) {
    @autoreleasepool {
        NSString *body = upitString(phase);
        if (total > 0) {
            body = [NSString stringWithFormat:@"%@ (%lld of %lld bytes)", body, processed, total];
        }
        if (atomic_load(&upitNotificationsAuthorized) == 1) {
            upitPostNotification(@"Upit", body, @"com.hungnth.upit.active");
        } else {
            upitUpdateFallback(body, processed, total);
        }
    }
}

void upitFeedbackComplete(const char *summary, int copy, int retry, int openDesktop) {
    @autoreleasepool {
        upitStopFallback();
        atomic_store(&upitAction, 0);
        upitRegisterTerminalCategory(copy != 0, retry != 0, openDesktop != 0);
        upitPostNotification(@"Upit", upitString(summary), @"com.hungnth.upit.terminal");
    }
}

static int upitRunAlert(NSString *message, BOOL copy, BOOL retry, BOOL openDesktop) {
    UpitAlertRequest *request = [UpitAlertRequest new];
    request.message = message;
    request.copyAction = copy;
    request.retryAction = retry;
    request.openDesktopAction = openDesktop;
    upitRunOnMainThread(request, @selector(runOnMainThread:), nil, YES);
    return request.result;
}

int upitFeedbackAlertWithActions(const char *summary, int copy, int retry, int openDesktop) {
    @autoreleasepool {
        upitStopFallback();
        return upitRunAlert(upitString(summary), copy != 0, retry != 0, openDesktop != 0);
    }
}

void upitFeedbackAlert(const char *message) {
    @autoreleasepool {
        upitStopFallback();
        upitRunAlert(upitString(message), NO, NO, NO);
    }
}

int upitFeedbackTakeAction(void) {
    return atomic_exchange(&upitAction, 0);
}

int upitFeedbackNotificationsAvailable(void) {
    return atomic_load(&upitNotificationsAuthorized);
}

void upitFeedbackClose(void) {
    upitStopFallback();
    atomic_store(&upitAction, 0);
}

void upitFeedbackPrepareEventLoop(void) {
    atomic_store(&upitStopEventLoop, false);
}

void upitFeedbackRunEventLoop(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    [NSApp finishLaunching];
    while (!atomic_load(&upitStopEventLoop)) {
        @autoreleasepool {
            NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:0.05];
            NSEvent *event = [NSApp nextEventMatchingMask:NSEventMaskAny
                                                untilDate:deadline
                                                   inMode:NSDefaultRunLoopMode
                                                  dequeue:YES];
            if (event != nil) {
                [NSApp sendEvent:event];
            }
            [[NSRunLoop currentRunLoop] runMode:NSDefaultRunLoopMode beforeDate:deadline];
        }
    }
}

void upitFeedbackStopEventLoop(void) {
    atomic_store(&upitStopEventLoop, true);
}
