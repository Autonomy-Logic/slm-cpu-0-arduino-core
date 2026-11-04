/*
 * Compile check for FACTS:stm32:slm_cpu_0: the network API as the OpenPLC
 * runtime uses it plus the bootloader API. Runs too: on TCP 502 it answers
 * 0x4C (then enters the bootloader), 0x47 and 0x48 as the runtime does and echoes
 * anything else; it replies on UDP 33333.
 */
#include <SPI.h>
#include <Ethernet.h>
#include <EthernetUdp.h>
#include <IPAddress.h>
#include <slm_cpu_0.h>

EthernetServer server(502);
EthernetUDP udp;
static uint32_t started;
static bool stable;

void setup()
{
  // The MAC argument is ignored; the platform uses the identity sector's.
  uint8_t mac[] = {0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x01};
  uint8_t ip[] = {192, 168, 1, 50};
  uint8_t dns[] = {192, 168, 1, 1};
  uint8_t gateway[] = {192, 168, 1, 1};
  uint8_t subnet[] = {255, 255, 255, 0};
  Ethernet.begin(mac, IPAddress(ip), IPAddress(dns), IPAddress(gateway), IPAddress(subnet));
  server.begin();
  udp.begin(33333);

  slm_cpu_0_cfg_net_t net = {};
  IPAddress a = Ethernet.localIP();
  for (int i = 0; i < 4; i++) {
    net.ip[i] = a[i];
    net.mask[i] = subnet[i];
    net.gw[i] = gateway[i];
    net.dns[i] = dns[i];
  }
  slm_cpu_0_remember_network(&net);
  started = millis();
}

// Modbus TCP reply: the request's MBAP header with a new length, then fc, 0x7E (success), data.
static void replyStatus(EthernetClient &client, const uint8_t *req, uint8_t fc, const uint8_t *data, uint8_t len)
{
  uint8_t out[7 + 2 + 32];
  if (len > 32) len = 32;
  memcpy(out, req, 7);
  uint16_t follow = (uint16_t)(1 + 2 + len);
  out[4] = (uint8_t)(follow >> 8);
  out[5] = (uint8_t)follow;
  out[7] = fc;
  out[8] = 0x7E;
  if (len) memcpy(out + 9, data, len);
  client.write(out, (size_t)(9 + len));
}

void loop()
{
  EthernetClient client = server.available();
  if (client && client.connected()) {
    uint8_t buf[260];
    int n = 0;
    while (client.available() && n < (int)sizeof(buf)) {
      buf[n++] = client.read();
    }
    // FC 0x4C with the magic B0 07 10 AD: answer 4C 7E, then hand over to the bootloader
    if (n >= 12 && buf[7] == 0x4C && buf[8] == 0xB0 && buf[9] == 0x07 && buf[10] == 0x10 && buf[11] == 0xAD) {
      replyStatus(client, buf, 0x4C, nullptr, 0);
      client.stop();
      uint8_t ip[4], mask[4], gw[4];
      IPAddress a = Ethernet.localIP(), m = Ethernet.subnetMask(), g = Ethernet.gatewayIP();
      for (int i = 0; i < 4; i++) {
        ip[i] = a[i];
        mask[i] = m[i];
        gw[i] = g[i];
      }
      slm_cpu_0_enter_bootloader(SLM_CPU_0_REASON_UPLOAD, ip, mask, gw);
    }
    // FC 0x47: runtime version, answered as 47 7E <text>
    if (n >= 8 && buf[7] == 0x47) {
      static const char version[] = "smoke";
      replyStatus(client, buf, 0x47, (const uint8_t *)version, sizeof(version) - 1);
      return;
    }
    // FC 0x48: device id, answered as 48 7E <length 0> (no licence identity)
    if (n >= 8 && buf[7] == 0x48) {
      static const uint8_t noId[] = {0};
      replyStatus(client, buf, 0x48, noId, sizeof(noId));
      return;
    }
    client.write(buf, n);
  }

  if (udp.parsePacket() > 0) {
    uint8_t mac[6] = {0};
    Ethernet.macAddress(mac);
    char reply[96];
    int len = snprintf(reply, sizeof(reply), "{\"device\":\"SLM-CPU-0\",\"mac\":\"%02x:%02x:%02x:%02x:%02x:%02x\"}",
                       mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
    udp.beginPacket(udp.remoteIP(), udp.remotePort());
    udp.write((const uint8_t *)reply, (size_t)len);
    udp.endPacket();
  }

  if (!stable && millis() - started >= SLM_CPU_0_APP_STABLE_MS) {
    slm_cpu_0_app_stable();
    stable = true;
  }
  digitalWrite(LED_RUN, (millis() / 500) & 1);
}
