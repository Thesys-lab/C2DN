cmake_minimum_required(VERSION 3.2)
project(CDNSimulator)
set(CDNSimulator_VERSION_MAJOR 0)
set(CDNSimulator_VERSION_MINOR 1)
set(CDNSimulator_VERSION_REVISION 0)


SET(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++0x")


# header
include_directories(libCacheSim/include)

include_directories(src/include)
include_directories(src/utils/include)
include_directories(tests/include)


# set CXX options
SET(CMAKE_CXX_FLAGS_DEBUG "$ENV{CXXFLAGS} -O0 -Wall -g -ggdb3 -DLOGLEVEL=6")
SET(CMAKE_CXX_FLAGS_RELEASE "$ENV{CXXFLAGS} -O3 -ggdb3 -Wall")


# find lib
# export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig:/home/ubuntu/software/lib/x86_64-linux-gnu/pkgconfig/
set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${CMAKE_SOURCE_DIR}/cmake/Modules/")
message(${CMAKE_MODULE_PATH})

find_package(GLib "2.40" REQUIRED)
include_directories(${GLib_INCLUDE_DIRS})
message(STATUS "gLib found? " ${GLib_FOUND} ", library " ${GLib_LIBRARY} ", header " ${GLib_INCLUDE_DIRS})

find_package(Boost COMPONENTS program_options REQUIRED)  #COMPONENTS program_options
include_directories(${Boost_INCLUDE_DIRS})
message(STATUS "boost found? " ${Boost_FOUND} ", library " ${Boost_LIBRARIES} ", header " ${Boost_INCLUDE_DIRS})

add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/libCacheSim)

aux_source_directory(tests dirTests)


find_package(ortools CONFIG REQUIRED)
include_directories(${ortools_INCLUDE_DIRS})
message(STATUS "ortools found? " ${ortools_FOUND} ", library " ${ortools_LIBRARIES} ", header " ${ortools_INCLUDE_DIRS})


# After the following call, the CMake targets defined by or-tools
# will be defined and available to the rest of the build
#FetchContent_MakeAvailable(or-tools)
add_subdirectory(src)


# executable
add_executable(simulator src/simulator.cpp)
target_link_libraries(simulator simulatorLib cache evictionAlgo traceReader profiler dataStructure traceStat utils
        ortools::ortools ${Boost_LIBRARIES} ${GLib_LIBRARY} ${CMAKE_THREAD_LIBS_INIT} -lpthread)

# -ltcmalloc -fno-builtin-malloc -fno-builtin-calloc -fno-builtin-realloc -fno-builtin-free

