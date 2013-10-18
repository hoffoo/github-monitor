github-monitor
==============

check github in a shell script - pass a username or language for recent events.

Options:

```sh
  -u        username, get recent events
  -l        language, search for top projects created recently
  -c        cut text after this length of output
  -d        show duplicate events
  -m        max number of items to display
  -debug    write github response to /tmp/github-monitor.json
```

I use it in conky like so: ${execi 3600 /opt/bin/gohub -c 30 hoffoo}

Output:

```sh
burtbeckwith forked groovydev/twitter-bootstrap-grails-plugin
guelfey commented guelfey/go.dbus
apttap pushed hoffoo/jizzmatic
aleksclark forked ebp/omniauth-box
```

Or projects and number of watchers:
```sh
skynetservices/skydns 75
mijia/gopark 41
kayac/isucon3 26
mattetti/go-web-api-demo 20
llimllib/loglevel 20
jingweno/gotask 17
```
