cmake_minimum_required(VERSION 3.16)
project(wirechat_sdk_c C)

set(CMAKE_C_STANDARD 11)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

find_package(PkgConfig REQUIRED)
pkg_check_modules(LIBWEBSOCKETS REQUIRED libwebsockets)

add_library(wirechat STATIC
    src/wirechat.c
    src/jsmn.h
)
target_include_directories(wirechat PUBLIC
    ${LIBWEBSOCKETS_INCLUDE_DIRS}
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)
target_compile_options(wirechat PRIVATE ${LIBWEBSOCKETS_CFLAGS_OTHER})
target_link_libraries(wirechat PUBLIC ${LIBWEBSOCKETS_LIBRARIES})

add_executable(hello examples/hello.c)
target_link_libraries(hello PRIVATE wirechat)
target_include_directories(hello PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/src)

add_executable(join_chat examples/join_chat.c)
target_link_libraries(join_chat PRIVATE wirechat)
target_include_directories(join_chat PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/src)
