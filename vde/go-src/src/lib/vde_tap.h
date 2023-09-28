#ifndef VDE_TAP_H
#define VDE_TAP_H

#include <stdint.h>

uintptr_t vde_tap_plug(char *, char *);

void vde_tap_unplug(uintptr_t);

#endif
