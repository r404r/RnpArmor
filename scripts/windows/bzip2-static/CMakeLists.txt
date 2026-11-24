cmake_minimum_required(VERSION 3.20)
project(bzip2_static C)
if(NOT BZIP2_SRC)
  message(FATAL_ERROR "set BZIP2_SRC to the extracted bzip2 1.0.8 directory")
endif()
file(TO_CMAKE_PATH "${BZIP2_SRC}" BZIP2_SRC_NORM)
add_library(bz2 STATIC
  "${BZIP2_SRC_NORM}/blocksort.c"
  "${BZIP2_SRC_NORM}/huffman.c"
  "${BZIP2_SRC_NORM}/crctable.c"
  "${BZIP2_SRC_NORM}/randtable.c"
  "${BZIP2_SRC_NORM}/compress.c"
  "${BZIP2_SRC_NORM}/decompress.c"
  "${BZIP2_SRC_NORM}/bzlib.c"
)
target_include_directories(bz2 PUBLIC "${BZIP2_SRC_NORM}")
target_compile_definitions(bz2 PRIVATE _CRT_SECURE_NO_WARNINGS _CRT_NONSTDC_NO_DEPRECATE)
set_target_properties(bz2 PROPERTIES OUTPUT_NAME bz2)
install(TARGETS bz2 ARCHIVE DESTINATION lib)
install(FILES "${BZIP2_SRC_NORM}/bzlib.h" DESTINATION include)
