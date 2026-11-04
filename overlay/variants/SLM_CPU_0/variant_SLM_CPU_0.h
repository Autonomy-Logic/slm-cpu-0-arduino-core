/*
 *******************************************************************************
 * SLM-CPU-0 (STM32F407VET6, LAN8742 PHY over RMII).
 *
 * Pin numbering is that of the STM32duino generic F407V(E-G)T variant, from
 * which this file is derived. Copyright (c) 2020, STMicroelectronics.
 * All rights reserved. Licensed under the BSD 3-Clause license:
 * opensource.org/licenses/BSD-3-Clause
 *******************************************************************************
 */
#pragma once

/*----------------------------------------------------------------------------
 *        STM32 pins number
 *----------------------------------------------------------------------------*/
#define PA0                     PIN_A0
#define PA1                     PIN_A1
#define PA2                     PIN_A2
#define PA3                     PIN_A3
#define PA4                     PIN_A4
#define PA5                     PIN_A5
#define PA6                     PIN_A6
#define PA7                     PIN_A7
#define PA8                     8
#define PA9                     9
#define PA10                    10
#define PA11                    11
#define PA12                    12
#define PA13                    13
#define PA14                    14
#define PA15                    15
#define PB0                     PIN_A8
#define PB1                     PIN_A9
#define PB2                     18
#define PB3                     19
#define PB4                     20
#define PB5                     21
#define PB6                     22
#define PB7                     23
#define PB8                     24
#define PB9                     25
#define PB10                    26
#define PB11                    27
#define PB12                    28
#define PB13                    29
#define PB14                    30
#define PB15                    31
#define PC0                     PIN_A10
#define PC1                     PIN_A11
#define PC2                     PIN_A12
#define PC3                     PIN_A13
#define PC4                     PIN_A14
#define PC5                     PIN_A15
#define PC6                     38
#define PC7                     39
#define PC8                     40
#define PC9                     41
#define PC10                    42
#define PC11                    43
#define PC12                    44
#define PC13                    45
#define PC14                    46
#define PC15                    47
#define PD0                     48
#define PD1                     49
#define PD2                     50
#define PD3                     51
#define PD4                     52
#define PD5                     53
#define PD6                     54
#define PD7                     55
#define PD8                     56
#define PD9                     57
#define PD10                    58
#define PD11                    59
#define PD12                    60
#define PD13                    61
#define PD14                    62
#define PD15                    63
#define PE0                     64
#define PE1                     65
#define PE2                     66
#define PE3                     67
#define PE4                     68
#define PE5                     69
#define PE6                     70
#define PE7                     71
#define PE8                     72
#define PE9                     73
#define PE10                    74
#define PE11                    75
#define PE12                    76
#define PE13                    77
#define PE14                    78
#define PE15                    79
#define PH0                     80
#define PH1                     81

// Alternate pins number
#define PA0_ALT1                (PA0  | ALT1)
#define PA0_ALT2                (PA0  | ALT2)
#define PA1_ALT1                (PA1  | ALT1)
#define PA1_ALT2                (PA1  | ALT2)
#define PA2_ALT1                (PA2  | ALT1)
#define PA2_ALT2                (PA2  | ALT2)
#define PA3_ALT1                (PA3  | ALT1)
#define PA3_ALT2                (PA3  | ALT2)
#define PA4_ALT1                (PA4  | ALT1)
#define PA5_ALT1                (PA5  | ALT1)
#define PA6_ALT1                (PA6  | ALT1)
#define PA7_ALT1                (PA7  | ALT1)
#define PA7_ALT2                (PA7  | ALT2)
#define PA7_ALT3                (PA7  | ALT3)
#define PA15_ALT1               (PA15 | ALT1)
#define PB0_ALT1                (PB0  | ALT1)
#define PB0_ALT2                (PB0  | ALT2)
#define PB1_ALT1                (PB1  | ALT1)
#define PB1_ALT2                (PB1  | ALT2)
#define PB3_ALT1                (PB3  | ALT1)
#define PB4_ALT1                (PB4  | ALT1)
#define PB5_ALT1                (PB5  | ALT1)
#define PB8_ALT1                (PB8  | ALT1)
#define PB9_ALT1                (PB9  | ALT1)
#define PB14_ALT1               (PB14 | ALT1)
#define PB14_ALT2               (PB14 | ALT2)
#define PB15_ALT1               (PB15 | ALT1)
#define PB15_ALT2               (PB15 | ALT2)
#define PC0_ALT1                (PC0  | ALT1)
#define PC0_ALT2                (PC0  | ALT2)
#define PC1_ALT1                (PC1  | ALT1)
#define PC1_ALT2                (PC1  | ALT2)
#define PC2_ALT1                (PC2  | ALT1)
#define PC2_ALT2                (PC2  | ALT2)
#define PC3_ALT1                (PC3  | ALT1)
#define PC3_ALT2                (PC3  | ALT2)
#define PC4_ALT1                (PC4  | ALT1)
#define PC5_ALT1                (PC5  | ALT1)
#define PC6_ALT1                (PC6  | ALT1)
#define PC7_ALT1                (PC7  | ALT1)
#define PC8_ALT1                (PC8  | ALT1)
#define PC9_ALT1                (PC9  | ALT1)
#define PC10_ALT1               (PC10 | ALT1)
#define PC11_ALT1               (PC11 | ALT1)

#define NUM_DIGITAL_PINS        82
#define NUM_ANALOG_INPUTS       16

/*----------------------------------------------------------------------------
 *        SLM-CPU-0 board signals
 *----------------------------------------------------------------------------*/
// Front panel LEDs, both active low
#define LED_RUN                 PE14
#define LED_CPU                 PE13
#ifndef LED_BUILTIN
  #define LED_BUILTIN           LED_RUN
#endif

// LAN8742 nRST, active low
#define PHY_RESET_PIN           PE11

// Slot backplane: SPI1 in slave mode plus the slot bus control lines
#define SLM_CPU_0_IO_SPI_NSS        PA4
#define SLM_CPU_0_IO_SPI_SCK        PA5
#define SLM_CPU_0_IO_SPI_MISO       PA6
#define SLM_CPU_0_IO_SPI_MOSI       PB5
#define SLM_CPU_0_IO_SSELN1         PB1
#define SLM_CPU_0_IO_DISC_OUT       PC0
#define SLM_CPU_0_IO_EN_BUFFN       PC8
#define SLM_CPU_0_IO_RESET_N        PC11
#define SLM_CPU_0_IO_ADDR0          PC12
#define SLM_CPU_0_IO_ADDR1          PC13
#define SLM_CPU_0_IO_ADDR2          PC14
#define SLM_CPU_0_IO_ADDR3          PC15
#define SLM_CPU_0_IO_SPARE_IN       PB7
#define SLM_CPU_0_IO_SPARE          PB8
#define SLM_CPU_0_TP1               PB6

// RMII (PA1 PA2 PA7 PB11 PB12 PB13 PC1 PC4 PC5) and JTAG (PA13 PA14 PA15
// PB3 PB4) are wired on the board; the defaults below avoid them.

// On-board user button
#ifndef USER_BTN
  #define USER_BTN              PNUM_NOT_DEFINED
#endif

// SPI definitions: none. SPI1 is the slot backplane bus (SPI slave); with no
// pins defined, SPI.begin() leaves it alone.
#ifndef PIN_SPI_SS
  #define PIN_SPI_SS            PNUM_NOT_DEFINED
#endif
#ifndef PIN_SPI_SS1
  #define PIN_SPI_SS1           PNUM_NOT_DEFINED
#endif
#ifndef PIN_SPI_SS2
  #define PIN_SPI_SS2           PNUM_NOT_DEFINED
#endif
#ifndef PIN_SPI_SS3
  #define PIN_SPI_SS3           PNUM_NOT_DEFINED
#endif
#ifndef PIN_SPI_MOSI
  #define PIN_SPI_MOSI          PNUM_NOT_DEFINED
#endif
#ifndef PIN_SPI_MISO
  #define PIN_SPI_MISO          PNUM_NOT_DEFINED
#endif
#ifndef PIN_SPI_SCK
  #define PIN_SPI_SCK           PNUM_NOT_DEFINED
#endif

// I2C definitions: I2C3 on two unconnected pins
#ifndef PIN_WIRE_SDA
  #define PIN_WIRE_SDA          PC9
#endif
#ifndef PIN_WIRE_SCL
  #define PIN_WIRE_SCL          PA8
#endif

// Timer Definitions
// Use TIM6/TIM7 when possible as servo and tone don't need GPIO output pin
#ifndef TIMER_TONE
  #define TIMER_TONE            TIM6
#endif
#ifndef TIMER_SERVO
  #define TIMER_SERVO           TIM7
#endif
// 1 ms LwIP scheduler tick
#ifndef DEFAULT_ETHERNET_TIMER
  #define DEFAULT_ETHERNET_TIMER TIM14
#endif

// UART Definitions
// No serial connector. The default Serial uses USART1 on the unconnected
// PA9/PA10, off the generic variant's PA1 (ETH_REF_CLK).
#ifndef SERIAL_UART_INSTANCE
  #define SERIAL_UART_INSTANCE  1
#endif
#ifndef PIN_SERIAL_RX
  #define PIN_SERIAL_RX         PA10
#endif
#ifndef PIN_SERIAL_TX
  #define PIN_SERIAL_TX         PA9
#endif

// Ethernet: STM32Ethernet drives the MAC through the legacy F4 HAL ETH API
#define ETHERNET_RMII_MODE_CONFIGURATION 1
#if !defined(HAL_ETH_MODULE_DISABLED)
  #define HAL_ETH_LEGACY_MODULE_ENABLED
#endif

// Extra HAL modules
#if !defined(HAL_DAC_MODULE_DISABLED)
  #define HAL_DAC_MODULE_ENABLED
#endif

/*----------------------------------------------------------------------------
 *        Arduino objects - C++ only
 *----------------------------------------------------------------------------*/

#ifdef __cplusplus
  #ifndef SERIAL_PORT_MONITOR
    #define SERIAL_PORT_MONITOR   Serial
  #endif
  #ifndef SERIAL_PORT_HARDWARE
    #define SERIAL_PORT_HARDWARE  Serial
  #endif
#endif
