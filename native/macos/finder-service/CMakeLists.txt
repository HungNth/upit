cmake_minimum_required(VERSION 3.20)
project(UpitFinderService LANGUAGES OBJC)

if(NOT APPLE)
    message(FATAL_ERROR "UpitFinderService requires macOS")
endif()

set(VERSION "0.7.0")
set(BUILD_VERSION "1")
configure_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/Info.plist.in"
    "${CMAKE_CURRENT_BINARY_DIR}/Info.plist"
    @ONLY
)

add_executable(UpitFinderService MACOSX_BUNDLE main.m selection.m)
enable_testing()
add_executable(UpitFinderSelectionTest selection.m selection_test.m)
target_link_libraries(UpitFinderSelectionTest PRIVATE "-framework Foundation")
add_test(NAME UpitFinderSelectionTest COMMAND UpitFinderSelectionTest)
target_link_libraries(UpitFinderService PRIVATE "-framework Cocoa")
set_target_properties(UpitFinderService PROPERTIES
    MACOSX_BUNDLE_BUNDLE_NAME "Upit"
    MACOSX_BUNDLE_GUI_IDENTIFIER "com.hungnth.upit.finder-service"
    MACOSX_BUNDLE_BUNDLE_VERSION "${BUILD_VERSION}"
    MACOSX_BUNDLE_SHORT_VERSION_STRING "${VERSION}"
    MACOSX_BUNDLE_INFO_PLIST "${CMAKE_CURRENT_BINARY_DIR}/Info.plist"
)
