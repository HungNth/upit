cmake_minimum_required(VERSION 3.20)
project(upit-explorer-command LANGUAGES CXX)

add_library(upit-explorer-command SHARED explorer_command.cpp)
target_compile_features(upit-explorer-command PRIVATE cxx_std_17)
target_compile_definitions(upit-explorer-command PRIVATE UNICODE _UNICODE WIN32_LEAN_AND_MEAN)
target_link_libraries(upit-explorer-command PRIVATE ole32 shell32 user32)
set_target_properties(upit-explorer-command PROPERTIES PREFIX "" OUTPUT_NAME "upit-explorer-command")
