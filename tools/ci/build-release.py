#!/usr/bin/env python3
"""
Build the FACTS:stm32 platform for the SLM-CPU-0, its uploader archives and
the board manager index that describes them.

The platform is STM32duino Arduino_Core_STM32 trimmed to the STM32F4 series,
plus overlay/ (boards.txt, the upload recipe and the SLM_CPU_0 variant) and
two bundled libraries: STM32duino LwIP and STM32Ethernet, renamed
SLM_CPU_0_LwIP and SLM_CPU_0_Ethernet and patched to take the MAC from the
board's identity sector. Upstream archives are fetched at the versions pinned
in upstream.json and checked by SHA-256.

    tools/ci/build-release.py --version 0.1.0 --out dist
    tools/ci/build-release.py --refresh-tools

Produces, under --out:
    FACTS-stm32-slm-cpu-0-<ver>.tar.bz2         the platform
    slm-cpu-0-netflash-<ver>-<host>.tar.gz      one per host
    package_FACTS_slm_cpu_0_index.json          the index

Uploader binaries must already be in <out>/tools/<host>/ (build-uploader.sh).
--no-netflash leaves the upload tool out, for compile-only test packages.
--refresh-tools copies the toolchain download entries for the versions named
in upstream.json from STMicroelectronics' index.

Standard library only.
"""
import argparse
import hashlib
import json
import os
import re
import shutil
import sys
import tarfile
import urllib.request
from pathlib import Path

HERE = Path(__file__).resolve().parent
REPO_ROOT = HERE.parent.parent
OVERLAY = REPO_ROOT / "overlay"
UPSTREAM_JSON = HERE / "upstream.json"

REPO = "Autonomy-Logic/slm-cpu-0-arduino-core"
PACKAGER = "FACTS"
ARCH = "stm32"
BOARD = "slm_cpu_0"
BOARD_NAME = "SLM-CPU-0"
VARIANT = "SLM_CPU_0"
LIB_PREFIX = "SLM_CPU_0"
PLATFORM_NAME = "FACTS SLM-CPU-0 (STM32F4)"
INDEX_NAME = "package_FACTS_slm_cpu_0_index.json"
SERIES = "STM32F4xx"
UPSTREAM_VARIANT = "variants/STM32F4xx/F407V(E-G)T_F417V(E-G)T"
NETFLASH = "slm-cpu-0-netflash"
# Every host the toolchain in upstream.json is published for.
HOSTS = ["x86_64-pc-linux-gnu", "aarch64-linux-gnu", "x86_64-apple-darwin", "arm64-apple-darwin", "i686-mingw32"]
# Shorter published tool names: the Windows toolchain include path must stay under MAX_PATH.
TOOL_NAMES = {"xpack-arm-none-eabi-gcc": "arm-gcc"}
DEFAULT_MTIME = 1767225600  # 2026-01-01T00:00:00Z


def fail(msg):
    sys.exit(f"build-release.py: {msg}")


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def fetch(url, dest, sha256):
    if dest.exists() and sha256_file(dest) == sha256:
        return dest
    print(f"downloading {url}")
    dest.parent.mkdir(parents=True, exist_ok=True)
    part = dest.with_name(dest.name + ".part")
    with urllib.request.urlopen(url) as r, open(part, "wb") as f:
        shutil.copyfileobj(r, f)
    got = sha256_file(part)
    if got != sha256:
        part.unlink()
        fail(f"{url}: SHA-256 is {got}, upstream.json pins {sha256}")
    part.replace(dest)
    return dest


def extract(archive, root, dest):
    """Extract `archive` and move its top-level folder `root` to `dest`."""
    tmp = dest.with_name(dest.name + ".extract")
    shutil.rmtree(tmp, ignore_errors=True)
    tmp.mkdir(parents=True)
    with tarfile.open(archive) as t:
        t.extractall(tmp, filter="data")
    if not (tmp / root).is_dir():
        fail(f"{archive.name}: no top-level folder {root}")
    shutil.rmtree(dest, ignore_errors=True)
    (tmp / root).rename(dest)
    shutil.rmtree(tmp)


def remove(base, rel):
    p = base / rel
    if p.is_dir() and not p.is_symlink():
        shutil.rmtree(p)
    elif p.exists() or p.is_symlink():
        p.unlink()
    else:
        fail(f"expected {p.relative_to(base.parent)} in the upstream archive")


def patch(path, old, new, count=1):
    text = path.read_text(encoding="utf-8")
    found = text.count(old)
    if found != count:
        fail(f"patch {path.name}: expected {count} match(es) of {old!r}, found {found}")
    path.write_text(text.replace(old, new), encoding="utf-8")


def sub_line(text, pattern, repl, what):
    text, n = re.subn(pattern, repl, text, flags=re.M)
    if n != 1:
        fail(f"platform.txt: expected one {what}, found {n}")
    return text


def check_includes(src, banned):
    """Fail if any #include under `src` still names a renamed header."""
    pat = re.compile(r'^\s*#\s*include\s*[<"](?:.*/)?(' + "|".join(map(re.escape, banned)) + r')[>"]', re.M)
    for p in sorted(src.rglob("*")):
        if p.suffix in (".h", ".c", ".cpp") and pat.search(p.read_text(encoding="utf-8", errors="replace")):
            fail(f"{p}: still includes a renamed header")


# ---- core -------------------------------------------------------------------

def trim_core(core):
    """Keep what an STM32F407 build needs; the full core is ~650 MB unpacked."""
    for rel in ("CI", "cmake", "README_CMAKE.md", "package.json", "tools", ".editorconfig", "variants", "debugger",
                "system/Middlewares/OpenAMP", "system/Middlewares/ST/STM32_USB_Host_Library",
                "libraries/VirtIO", "libraries/SubGhz", "libraries/CMSIS_DSP"):
        remove(core, rel)
    for d in sorted((core / "system").iterdir()):
        if d.is_dir() and d.name.startswith("STM32") and d.name != SERIES:
            remove(core, d.relative_to(core))
    drivers = core / "system/Drivers"
    for d in sorted(drivers.iterdir()):
        if d.name.endswith("_HAL_Driver") and d.name != f"{SERIES}_HAL_Driver":
            remove(core, d.relative_to(core))
    st = drivers / "CMSIS/Device/ST"
    for d in sorted(st.iterdir()):
        if d.is_dir() and d.name != SERIES:
            remove(core, d.relative_to(core))
    # stm32f4xx.h includes only the STM32F407 device header.
    keep = {"stm32f4xx.h", "stm32f407xx.h", "system_stm32f4xx.h"}
    for f in sorted((st / SERIES / "Include").iterdir()):
        if f.name not in keep:
            f.unlink()
    for f in sorted((st / SERIES / "Source/Templates/gcc").iterdir()):
        if f.name != "startup_stm32f407xx.s":
            f.unlink()
    for p in sorted(core.rglob("CMakeLists.txt")):
        p.unlink()


def patch_platform(path, version):
    text = path.read_text(encoding="utf-8")
    text = sub_line(text, r"^name=.*$", f"name={PLATFORM_NAME}", "name=")
    text = sub_line(text, r"^version=.*$", f"version={version}", "version=")
    # Tools the platform does not ship: debugger, DSP libraries
    for var in ("openocd_dir", "cmsis_dsp", "cmsis_nn"):
        text = sub_line(text, rf"^{var}=.*\n", "", f"{var}=")
    dsp = ' "-I{cmsis_dsp}/Include" "-I{cmsis_dsp}/PrivateInclude"'
    if text.count(dsp) != 1:
        fail("platform.txt: CMSIS-DSP include flags not found")
    text = text.replace(dsp, "")
    for old, new in TOOL_NAMES.items():
        ref = f"{{runtime.tools.{old}-"
        if text.count(ref) != 1:
            fail(f"platform.txt: expected one reference to the {old} tool")
        text = text.replace(ref, f"{{runtime.tools.{new}-")
    head, sep, _ = text.partition("\n# Uploader tool\n")
    if not sep:
        fail("platform.txt: uploader section not found")
    tail = (OVERLAY / "platform.upload.txt").read_text(encoding="utf-8")
    path.write_text(head.rstrip("\n") + "\n\n\n" + tail, encoding="utf-8")


def apply_overlay(core, variant_files, version):
    shutil.copyfile(OVERLAY / "boards.txt", core / "boards.txt")
    shutil.copytree(OVERLAY / "variants", core / "variants")
    variant = core / "variants" / VARIANT
    for name, data in variant_files.items():
        (variant / name).write_bytes(data)
    patch_platform(core / "platform.txt", version)
    (core / "package_version.txt").write_text(version + "\n", encoding="utf-8")


# ---- bundled libraries --------------------------------------------------------

LWIP_PROPERTIES = f"""\
name={LIB_PREFIX}_LwIP
version=2.2.1
author=Adam Dunkels <adam@sics.se>
maintainer=Autonomy Logic (SLM-CPU-0 build of STM32duino LwIP)
sentence=A Lightweight TCP/IP stack, bundled for the SLM-CPU-0.
paragraph=STM32duino LwIP 2.2.1 under a unique name. Pulled in by {LIB_PREFIX}_Ethernet.
category=Communication
url=https://github.com/stm32duino/LwIP
architectures=stm32
includes={LIB_PREFIX}_LwIP.h
"""

ETHERNET_PROPERTIES = f"""\
name={LIB_PREFIX}_Ethernet
version=1.5.0
author=Various
maintainer=Autonomy Logic (SLM-CPU-0 build of STM32duino STM32Ethernet)
sentence=Ethernet for the SLM-CPU-0, reached through <Ethernet.h>.
paragraph=STM32duino STM32Ethernet 1.5.0 under a unique name, with the MAC taken from the board's identity sector.
category=Communication
url=https://github.com/stm32duino/STM32Ethernet
architectures=stm32
includes={LIB_PREFIX}_Ethernet.h
"""


def build_lwip(src, dest):
    shutil.copytree(src, dest)
    (dest / "src/LwIP.h").rename(dest / f"src/{LIB_PREFIX}_LwIP.h")
    (dest / "library.properties").write_text(LWIP_PROPERTIES, encoding="utf-8")
    check_includes(dest / "src", ["LwIP.h"])


def build_ethernet(src, dest):
    shutil.copytree(src, dest)
    for rel in (".github", "examples", "library.json"):
        remove(dest, rel)
    s = dest / "src"
    eth_h, udp_h = f"{LIB_PREFIX}_Ethernet.h", f"{LIB_PREFIX}_EthernetUdp.h"
    (s / "STM32Ethernet.h").rename(s / eth_h)
    (s / "EthernetUdp.h").rename(s / udp_h)
    for name in ("STM32Ethernet.cpp", "EthernetClient.cpp", "EthernetServer.cpp", "EthernetUdp.cpp"):
        patch(s / name, '#include "STM32Ethernet.h"', f'#include "{eth_h}"')
    patch(s / "Dhcp.h", '#include "EthernetUdp.h"', f'#include "{udp_h}"')
    patch(s / "Dns.cpp", '#include "EthernetUdp.h"', f'#include "{udp_h}"')
    patch(s / "Dns.h", "#include <EthernetUdp.h>", f'#include "{udp_h}"')

    h = s / eth_h
    # LwIP is found only through its marker header.
    patch(h, "#define ethernet_h\n", f"#define ethernet_h\n\n#include <{LIB_PREFIX}_LwIP.h>\n")
    patch(h, "    void MACAddress(uint8_t *mac_address);\n",
          "    void MACAddress(uint8_t *mac_address);\n"
          "    // Arduino Ethernet library spelling, used by the OpenPLC runtime\n"
          "    void macAddress(uint8_t *mac_address)\n"
          "    {\n"
          "      MACAddress(mac_address);\n"
          "    }\n")
    # DHCP timeout shortened to fit inside the watchdog window kept during setup().
    patch(h, "int begin(unsigned long timeout = 60000,", "int begin(unsigned long timeout = 20000,")
    patch(h, "int begin(uint8_t *mac_address, unsigned long timeout = 60000,",
          "int begin(uint8_t *mac_address, unsigned long timeout = 20000,")
    # On DHCP failure, fall back to the bootloader's default link-local address.
    cpp = s / "STM32Ethernet.cpp"
    patch(cpp, '#include "Dhcp.h"\n', '#include "Dhcp.h"\n#include "boot_shared.h"\n')
    patch(cpp, "  if (ret == 1) {\n    _dnsServerAddress = _dhcp->getDnsServerIp();\n  }\n",
          "  if (ret == 1) {\n    _dnsServerAddress = _dhcp->getDnsServerIp();\n  } else {\n"
          "    static const uint8_t ip[4] = SLM_CPU_0_DEFAULT_IP, mask[4] = SLM_CPU_0_DEFAULT_MASK;\n"
          "    begin(IPAddress(ip), IPAddress(mask), IPAddress(0, 0, 0, 0), IPAddress(0, 0, 0, 0));\n"
          "  }\n", count=2)

    eif = s / "utility/ethernetif.cpp"
    patch(eif, '#include "stm32_eth.h"\n', '#include "stm32_eth.h"\n#include "boot_shared.h"\n')
    patch(eif, "static void low_level_init(struct netif *netif)\n{\n  uint32_t regvalue;\n\n",
          "static void low_level_init(struct netif *netif)\n{\n  uint32_t regvalue;\n\n"
          "  /* SLM-CPU-0: MAC from the identity sector (or derived from the unique ID),\n"
          "   * whatever the sketch passed to Ethernet.begin(). */\n"
          "  slm_cpu_0_get_mac(macaddress);\n\n")

    # Arduino Ethernet 2.x accept(): each new connection is handed out once and
    # leaves the server's table, so the caller's stop() is its only owner.
    srv_h, srv_cpp = s / "EthernetServer.h", s / "EthernetServer.cpp"
    patch(srv_h, "    void accept(void);\n  public:\n    EthernetServer(uint16_t port = 80);\n",
          "    void reap(void);\n  public:\n    EthernetServer(uint16_t port = 80);\n"
          "    EthernetClient accept();\n")
    patch(srv_cpp, "void EthernetServer::accept()\n", "void EthernetServer::reap()\n")
    patch(srv_cpp, "  accept();\n", "  reap();\n", count=2)
    patch(srv_cpp, "EthernetClient EthernetServer::available()\n",
          "EthernetClient EthernetServer::accept()\n{\n  reap();\n\n"
          "  for (int n = 0; n < MAX_CLIENT; n++) {\n"
          "    struct tcp_struct *tcp = _tcp_client[n];\n"
          "    if (tcp != NULL && tcp->pcb != NULL && tcp->state == TCP_ACCEPTED) {\n"
          "      _tcp_client[n] = NULL;\n"
          "      return EthernetClient(tcp);\n"
          "    }\n  }\n\n"
          "  struct tcp_struct *none = NULL;\n  return EthernetClient(none);\n}\n\n"
          "EthernetClient EthernetServer::available()\n")
    cli_h, cli_cpp = s / "EthernetClient.h", s / "EthernetClient.cpp"
    patch(cli_h, "    virtual int available();\n",
          "    virtual int available();\n    virtual int availableForWrite();\n")
    patch(cli_cpp, "int EthernetClient::read()\n",
          "int EthernetClient::availableForWrite()\n{\n"
          "  if ((_tcp_client == NULL) || (_tcp_client->pcb == NULL) ||\n"
          "      ((_tcp_client->state != TCP_ACCEPTED) && (_tcp_client->state != TCP_CONNECTED))) {\n"
          "    return 0;\n  }\n  return tcp_sndbuf(_tcp_client->pcb);\n}\n\n"
          "int EthernetClient::read()\n")
    # Upstream reads this flag uninitialised when every slot is taken.
    patch(s / "utility/stm32_eth.cpp", "  uint8_t accepted;\n", "  uint8_t accepted = 0;\n")

    (dest / "library.properties").write_text(ETHERNET_PROPERTIES, encoding="utf-8")
    check_includes(s, ["STM32Ethernet.h", "EthernetUdp.h", "LwIP.h"])


# ---- archives and index ---------------------------------------------------------

def clean_tree(root):
    """Drop macOS metadata files so they never reach an archive."""
    for p in sorted(root.rglob("*")):
        if p.is_file() and (p.name.startswith("._") or p.name == ".DS_Store"):
            p.unlink()


def make_archive(src, archive, top, mtime):
    def normalize(ti):
        ti.uid = ti.gid = 0
        ti.uname = ti.gname = ""
        ti.mtime = mtime
        if ti.isdir():
            ti.mode = 0o755
        elif ti.isfile():
            ti.mode = 0o755 if ti.mode & 0o111 else 0o644
        return ti

    archive.parent.mkdir(parents=True, exist_ok=True)
    mode = "w:bz2" if archive.name.endswith(".tar.bz2") else "w:gz"
    # tarfile.add recurses in sorted order, so equal inputs give equal archives.
    with tarfile.open(archive, mode) as t:
        t.add(src, arcname=top, filter=normalize)


def asset(path, base_url):
    return {
        "url": f"{base_url}/{path.name}",
        "archiveFileName": path.name,
        "checksum": f"SHA-256:{sha256_file(path)}",
        "size": str(path.stat().st_size),
    }


def netflash_tool(out, version, base_url, mtime):
    """Pack <out>/tools/<host>/slm-cpu-0-netflash[.exe] into one archive per host."""
    systems = []
    stage = out / ".stage-tool"
    for host in HOSTS:
        exe = NETFLASH + (".exe" if "mingw32" in host else "")
        src = out / "tools" / host / exe
        if not src.is_file():
            fail(f"missing uploader for {host}: {src} (run tools/ci/build-uploader.sh {out / 'tools'})")
        shutil.rmtree(stage, ignore_errors=True)
        (stage / NETFLASH).mkdir(parents=True)
        dst = stage / NETFLASH / exe
        shutil.copyfile(src, dst)
        dst.chmod(0o755)
        archive = out / f"{NETFLASH}-{version}-{host}.tar.gz"
        make_archive(stage / NETFLASH, archive, NETFLASH, mtime)
        systems.append({"host": host, **asset(archive, base_url)})
    shutil.rmtree(stage, ignore_errors=True)
    return {"name": NETFLASH, "version": version, "systems": systems}


def check_upstream_hosts(up):
    for t in up["tools"]:
        if "systems" not in t:
            fail(f"upstream.json has no downloads for {t['name']} {t['version']}; run --refresh-tools")
        hosts = {s["host"] for s in t["systems"]}
        if hosts != set(HOSTS):
            fail(f"upstream.json: {t['name']} hosts {sorted(hosts)} differ from {sorted(HOSTS)}")


def build(args):
    if not re.fullmatch(r"\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?", args.version):
        fail(f"--version {args.version!r} is not a semantic version")
    base_url = (args.base_url or f"https://github.com/{REPO}/releases/download/v{args.version}").rstrip("/")
    up = json.loads(UPSTREAM_JSON.read_text(encoding="utf-8"))
    check_upstream_hosts(up)

    out = Path(args.out).resolve()
    work = Path(args.work).resolve()
    cache, stage = work / "cache", work / "stage"
    mtime = int(os.environ.get("SOURCE_DATE_EPOCH", DEFAULT_MTIME))

    src = {}
    for key, a in up["archives"].items():
        archive = fetch(a["url"], cache / f"{key}-{a['url'].rsplit('/', 1)[1]}", a["sha256"])
        src[key] = stage / "upstream" / key
        print(f"extracting {archive.name}")
        extract(archive, a["root"], src[key])

    core = stage / "platform"
    shutil.rmtree(core, ignore_errors=True)
    src["core"].rename(core)
    variant_files = {n: (core / UPSTREAM_VARIANT / n).read_bytes() for n in ("PeripheralPins.c", "PinNamesVar.h")}
    trim_core(core)
    apply_overlay(core, variant_files, args.version)
    build_lwip(src["lwip"], core / f"libraries/{LIB_PREFIX}_LwIP")
    build_ethernet(src["ethernet"], core / f"libraries/{LIB_PREFIX}_Ethernet")
    clean_tree(core)
    leaked = [p for p in core.rglob("*") if "slmbackplane" in p.name.lower()]
    if leaked:
        fail(f"the platform must not contain SLMBackplane: {leaked[0]}")

    out.mkdir(parents=True, exist_ok=True)
    for old in out.glob(f"{PACKAGER}-{ARCH}-*.tar.bz2"):
        old.unlink()
    archive = out / f"{PACKAGER}-{ARCH}-slm-cpu-0-{args.version}.tar.bz2"
    print(f"packing {archive.name}")
    make_archive(core, archive, f"{ARCH}-{args.version}", mtime)

    tools = [dict(t, name=TOOL_NAMES.get(t["name"], t["name"])) for t in up["tools"]]
    if not args.no_netflash:
        tools.append(netflash_tool(out, args.version, base_url, mtime))
    platform = {
        "name": PLATFORM_NAME,
        "architecture": ARCH,
        "version": args.version,
        "category": "Contributed",
        "help": {"online": f"https://github.com/{REPO}"},
        **asset(archive, base_url),
        "boards": [{"name": BOARD_NAME}],
        "toolsDependencies": [{"packager": PACKAGER, "name": t["name"], "version": t["version"]} for t in tools],
    }
    index = {"packages": [{
        "name": PACKAGER,
        "maintainer": "Autonomy Logic",
        "websiteURL": f"https://github.com/{REPO}",
        "email": "support@autonomylogic.com",
        "help": {"online": f"https://github.com/{REPO}"},
        "platforms": [platform],
        "tools": tools,
    }]}
    (out / INDEX_NAME).write_text(json.dumps(index, indent=2) + "\n", encoding="utf-8")

    print(f"{archive}: {platform['size']} bytes, sha256 {platform['checksum'][8:]}")
    print(f"{out / INDEX_NAME}: FQBN {PACKAGER}:{ARCH}:{BOARD}, served from {base_url}")


def refresh_tools():
    up = json.loads(UPSTREAM_JSON.read_text(encoding="utf-8"))
    with urllib.request.urlopen(up["st_index"]) as r:
        st = json.load(r)
    available = {(t["name"], t["version"]): t for p in st["packages"] for t in p.get("tools", [])}
    tools = []
    for t in up["tools"]:
        key = (t["name"], t["version"])
        if key not in available:
            fail(f"{t['name']} {t['version']} is not in {up['st_index']}")
        tools.append(available[key])
    up["tools"] = tools
    UPSTREAM_JSON.write_text(json.dumps(up, indent=2) + "\n", encoding="utf-8")
    print(f"updated {UPSTREAM_JSON.name}: " + ", ".join(f"{t['name']} {t['version']}" for t in tools))


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--version", help="platform and uploader version, e.g. 0.1.0")
    ap.add_argument("--out", default=str(REPO_ROOT / "dist"), help="output folder (default: dist/)")
    ap.add_argument("--work", default=str(REPO_ROOT / "build"), help="download cache and staging (default: build/)")
    ap.add_argument("--base-url", help="URL the files in --out are served from (default: the GitHub release)")
    ap.add_argument("--no-netflash", action="store_true", help="leave the upload tool out (compile-only package)")
    ap.add_argument("--refresh-tools", action="store_true", help="update upstream.json from STMicroelectronics' index")
    args = ap.parse_args()
    if args.refresh_tools:
        refresh_tools()
    elif args.version:
        build(args)
    else:
        ap.error("--version is required unless --refresh-tools is given")


if __name__ == "__main__":
    main()
