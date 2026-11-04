/*
 * slm_cpu_0_port.c - see slm_cpu_0.h. Also the flash access slm_cpu_0_boot.c
 * needs, restricted to the config log sector.
 */
#include <string.h>

#include "Arduino.h"
#include "slm_cpu_0.h"

int slm_cpu_0_port_flash_erase_sector(uint32_t sector)
{
    if (sector != SLM_CPU_0_CFG_SECTOR) return -1;

    FLASH_EraseInitTypeDef erase;
    memset(&erase, 0, sizeof(erase));
    erase.TypeErase = FLASH_TYPEERASE_SECTORS;
    erase.Sector = sector;
    erase.NbSectors = 1;
    erase.VoltageRange = FLASH_VOLTAGE_RANGE_3;
    uint32_t failed = 0;

    HAL_FLASH_Unlock();
    __HAL_FLASH_CLEAR_FLAG(FLASH_FLAG_EOP | FLASH_FLAG_OPERR | FLASH_FLAG_WRPERR |
                           FLASH_FLAG_PGAERR | FLASH_FLAG_PGPERR | FLASH_FLAG_PGSERR);
    HAL_StatusTypeDef st = HAL_FLASHEx_Erase(&erase, &failed);
    HAL_FLASH_Lock();
    return st == HAL_OK ? 0 : -1;
}

int slm_cpu_0_port_flash_program_word(uint32_t addr, uint32_t value)
{
    if (addr < SLM_CPU_0_CFG_BASE || addr > SLM_CPU_0_CFG_BASE + SLM_CPU_0_CFG_SIZE - 4u || (addr & 3u)) return -1;

    HAL_FLASH_Unlock();
    __HAL_FLASH_CLEAR_FLAG(FLASH_FLAG_EOP | FLASH_FLAG_OPERR | FLASH_FLAG_WRPERR |
                           FLASH_FLAG_PGAERR | FLASH_FLAG_PGPERR | FLASH_FLAG_PGSERR);
    HAL_StatusTypeDef st = HAL_FLASH_Program(FLASH_TYPEPROGRAM_WORD, addr, value);
    HAL_FLASH_Lock();
    return st == HAL_OK ? 0 : -1;
}

static void bkp_write(uint32_t idx, uint32_t value)
{
    __HAL_RCC_PWR_CLK_ENABLE();
    HAL_PWR_EnableBkUpAccess();
    (&RTC->BKP0R)[idx] = value;
}

void slm_cpu_0_enter_bootloader(uint32_t reason, const uint8_t ip[4], const uint8_t mask[4], const uint8_t gw[4])
{
    uint32_t a = ip != NULL ? slm_cpu_0_ip_pack(ip) : 0;
    uint32_t m = mask != NULL ? slm_cpu_0_ip_pack(mask) : 0;
    uint32_t g = gw != NULL ? slm_cpu_0_ip_pack(gw) : 0;

    __disable_irq();
    bkp_write(SLM_CPU_0_BKP_REASON, reason);
    bkp_write(SLM_CPU_0_BKP_IP, a);
    bkp_write(SLM_CPU_0_BKP_MASK, m);
    bkp_write(SLM_CPU_0_BKP_GW, g);
    bkp_write(SLM_CPU_0_BKP_CHECK, slm_cpu_0_mailbox_check(reason, a, m, g));
    bkp_write(SLM_CPU_0_BKP_MAGIC, SLM_CPU_0_MAILBOX_MAGIC);
    NVIC_SystemReset();
}

void slm_cpu_0_app_stable(void)
{
    bkp_write(SLM_CPU_0_BKP_BOOTCOUNT, 0);
}

int slm_cpu_0_remember_network(const slm_cpu_0_cfg_net_t *net)
{
    const slm_cpu_0_cfg_entry_t *e = slm_cpu_0_cfg_latest(SLM_CPU_0_CFG_NET);
    if (e != NULL && e->len == sizeof(*net) && memcmp((const void *)e->data, net, sizeof(*net)) == 0) return 0;
    return slm_cpu_0_cfg_append(SLM_CPU_0_CFG_NET, net, sizeof(*net));
}
