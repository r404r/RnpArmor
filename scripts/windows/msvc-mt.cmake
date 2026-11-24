# Static Microsoft C runtime. The portable directory then does not ship
# vcruntime140.dll. UCRT stays a Windows 10+ system library when the
# linker still imports it; dumpbin records the actual imports.
set(CMAKE_MSVC_RUNTIME_LIBRARY "MultiThreaded" CACHE STRING "" FORCE)
cmake_policy(SET CMP0091 NEW)
add_compile_options($<$<C_COMPILER_ID:MSVC>:/FS> $<$<CXX_COMPILER_ID:MSVC>:/FS>)
