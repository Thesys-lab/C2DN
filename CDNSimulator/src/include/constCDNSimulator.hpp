//
//  constCDNSimulator.cpp
//  CDNSimulator
//
//  Created by Juncheng on 7/11/17.
//  Copyright © 2017 Juncheng. All rights reserved.
//


#ifndef constCDNSimulator_HPP
#define constCDNSimulator_HPP

#ifdef __cplusplus
extern "C"
{
#endif

#include <stdio.h>
#include <glib.h>

#ifdef __cplusplus
}
#endif

typedef enum {
  full_obj = 1,
  chunk_obj = 2,
  unknown_obj_type = 3,

  invalid_obj = 4 
}obj_type_e;

typedef enum {
  akamai1b,
  akamai1bWithHost,
  akamai1bWithBucket
} trace_format_e;


#define MAX_EC_N 16
#define MAX_N_SERVER 128
#define MAX_N_UNAVAIL 2 


#endif /* constCDNSimulator_HPP */

