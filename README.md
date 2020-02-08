# dbus-inspect

Command line D-Bus inspector.

## Usecase

It's an alternative to [D-Feet](https://wiki.gnome.org/Apps/DFeet) which requires a graphical environment or a nicer-looking replacement of [`busctl introspect`](https://www.freedesktop.org/software/systemd/man/busctl.html).

## Installation

You can compile it from sources or install it using `go get`:

```bash
go get -u github.com/goautomotive/dbus-inspect
```

## Usage

List all available destinations on the system bus:

```
% dbus-inspect -system ⏎

fi.w1.wpa_supplicant1 611 /usr/bin/wpa_supplicant -u
org.bluez 977 /usr/lib/bluetooth/bluetoothd
org.freedesktop.Accounts 616 /usr/lib/accounts-daemon
org.freedesktop.Avahi 624 avahi-daemon: running [homebook.local
org.freedesktop.ColorManager 709 /usr/lib/colord
org.freedesktop.DBus 615 /usr/bin/dbus-daemon --system --address=systemd: --nofork --nopidfile --systemd-activation --syslog-only
...
``` 

Inspect a particular destination, say `org.bluez`:

```
% dbus-inspect -system org.bluez ⏎

/
  org.freedesktop.DBus.Introspectable
    Methods
      Introspect() → (xml String)
  org.freedesktop.DBus.ObjectManager
    Methods
      GetManagedObjects() → (objects Dict{Object, Dict{String, Dict{String, Variant}}})
    Signals
      InterfacesAdded(object Object, interfaces Dict{String, Dict{String, Variant}})
      InterfacesRemoved(object Object, interfaces Array[String])

/org/bluez
  org.freedesktop.DBus.Introspectable
    Methods
      Introspect() → (xml String)
  org.bluez.AgentManager1
    Methods
      RegisterAgent(agent Object, capability String) → ()
      UnregisterAgent(agent Object) → ()
      RequestDefaultAgent(agent Object) → ()
  org.bluez.ProfileManager1
    Methods
      RegisterProfile(profile Object, UUID String, options Dict{String, Variant}) → ()
      UnregisterProfile(profile Object) → ()
...
```

For additional help use `-h` or `-help` 

## Contributing

All contributions are welcome! Just fill in an issue or submit a pull request.
