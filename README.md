github-monitor
==============

check github events in a shell script - pass a username to see recent events.

Options:

```sh
  -c=0: cut after this length of output
  -m=5: max number of events to display
  -d=false: debug - read from file instead of request
```

I use it in conky like so: ${execi 3600 /opt/bin/gohub -c 30 hoffoo}

Output:

```sh
burtbeckwith forked groovydev/twitter-bootstrap-grails-plugin
guelfey commented guelfey/go.dbus
apttap pushed hoffoo/jizzmatic
aleksclark forked ebp/omniauth-box
```

Note that github allows 60 request per hour
