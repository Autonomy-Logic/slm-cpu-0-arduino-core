/*
 * slm_cpu_0.h - board services around the network bootloader for sketches
 * (the OpenPLC HAL). Flash layout and record formats are in boot_shared.h.
 */
#ifndef SLM_CPU_0_H
#define SLM_CPU_0_H

#include <stdint.h>

#include "boot_shared.h"

#ifdef __cplusplus
extern "C" {
#endif

/* Reset into the network bootloader and stay there, listening on the given
 * address (NULL or 0.0.0.0 lets it choose). `reason` is SLM_CPU_0_REASON_*.
 * Does not return. */
void slm_cpu_0_enter_bootloader(uint32_t reason, const uint8_t ip[4], const uint8_t mask[4],
                                const uint8_t gw[4]) __attribute__((noreturn));

/* Clear the watchdog reset counter. Call once the application has run for
 * SLM_CPU_0_APP_STABLE_MS; SLM_CPU_0_CRASH_LOOP_LIMIT uncleared resets keep
 * the bootloader waiting for a new image. */
void slm_cpu_0_app_stable(void);

/* Record the network settings in the config log for the bootloader. Writes
 * flash only when they differ from the last record. Returns 0 on success. */
int slm_cpu_0_remember_network(const slm_cpu_0_cfg_net_t *net);

#ifdef __cplusplus
}
#endif

#endif /* SLM_CPU_0_H */
