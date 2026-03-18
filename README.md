# jitterbugs

A local-first camera utility for previewing and recording video to local storage.

## What this is

jitterbugs is a simple, privacy-respecting camera/recorder application. It is designed for personal use where the user controls what is captured, stored, and deleted.

**Goals:**
- Camera preview and recording to local storage
- User-controlled start/stop
- Local file management (retain/delete/export)
- Clear, prominent UI indicators when recording is active

**Non-goals:**
- No facial recognition or identity matching
- No emotion detection or behavioral inference
- No cloud uploads by default
- No background or hidden recording

See [docs/SAFETY-SCOPE-POLICY.md](docs/SAFETY-SCOPE-POLICY.md) for the full scope and data-handling policy.

## Building & Packaging

See [docs/build.md](docs/build.md) for instructions on building the desktop
application locally (Linux binary, `.deb` package, Windows `.exe`) and for
an overview of the CI pipeline.

## License

This project is licensed under the **GNU General Public License v3.0 (GPLv3)**.
See [LICENSE](LICENSE) for the full license text.

## Trademark

The names "Jitterbug" and "Jitterbugs" and any associated logos are trademarks of the
project owners.  The GPLv3 license grants you the right to use and redistribute the code,
but does **not** grant trademark rights.  See [TRADEMARK.md](TRADEMARK.md) for details on
what is and is not permitted for forks and derivative works.

## Security Posture

Jitterbugs is designed around a **VPN-only / no inbound ports** security model.  Cameras
and recording devices are kept on a private network; remote access requires the owner to
connect via an authenticated VPN tunnel—no cloud relay with shared credentials, no UPnP,
no open port forwarding.

See [docs/SECURITY_ARCHITECTURE.md](docs/SECURITY_ARCHITECTURE.md) for a full description
of the recommended network topology, access controls, and operational guidelines.
