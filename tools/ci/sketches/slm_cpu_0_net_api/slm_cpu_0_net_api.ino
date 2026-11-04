/*
 * Compile check for the Arduino Ethernet 2.x server API the OpenPLC runtime's
 * OPC UA and S7 servers use: EthernetServer::accept() and
 * EthernetClient::availableForWrite().
 */
#include <SPI.h>
#include <Ethernet.h>

EthernetServer server(4840);
EthernetClient held;

void setup()
{
  uint8_t mac[] = {0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x02};
  Ethernet.begin(mac, IPAddress(192, 168, 1, 51));
  server.begin();
}

void loop()
{
  if (!held) {
    held = server.accept();
  }
  if (held && !held.connected()) {
    held.stop();
  }
  if (held && held.available() > 0 && held.availableForWrite() > 0) {
    held.write((uint8_t)held.read());
  }
}
