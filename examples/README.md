### WIP: A (very) brief how-to for a simple Waitron set-up

These steps assume you already have have a DHCP server handing out IP addresses.  Waitron and Pixiecore work along-side _existing_ DHCP servers.  It's possible to hand out temporary, local IPs with DHCP and then switch to public v4 or v6 for the rest of the install process.

As long as your DHCP server is handing out addresses that will allow you to reach your local pixiecore installation, things should just work.  If you're planning to do a full OS install from netboot, which is what the examples in the repo attempt, you'll only need to ensure that the IP settings in dns02.example.com.yml can reach the outside world.

In the examples/machines directory, make sure to update dns02.example.com.yml with the MAC of your installing machine and the IP details you'd like it to have. If the server running Waitron has access to run IPMI commands on your target device, set the IPMI details in dns02.example.com.yml and uncomment the ipmitool lines in the examples/messages files.

Also, change `SOME_PASSWORD_THAT_YOU_SHOULD_CHANGE` in the examples/templates/preseed.j2 file. :)

# pixiecore

You'll need to run pixiecore on a machine in the same network where you plan to boot your new machine.

First, build pixiecore:

```
(GOROOT=/usr/local/go; cd /tmp/ \
        && git clone https://github.com/google/netboot.git \
        && mkdir -p $GOROOT/src/go.universe.tf \
        && ln -s /tmp/netboot  $GOROOT/src/go.universe.tf/netboot \
        && go get golang.org/x/net/ipv4 \
        && go get golang.org/x/net/ipv6 \
        && go get golang.org/x/net/bpf \
        && go get golang.org/x/crypto/nacl/secretbox \
        && go get github.com/spf13/cobra \
        && go get github.com/spf13/viper \
        && cd netboot/cmd/pixiecore \
    && CGO_ENABLED=0 go build -o pixiecore main.go \
    && mv pixiecore /usr/local/bin/pixiecore)
```

Next, run pixiecore:

```
pixiecore api http://my_waitron_location --dhcp-no-bind --log-timestamps --debug --port 5058 --status-port 5058
```

# Waitron

Next, run an instance of Waitron at any location that would be reachable by the machine running pixiecore via http:

```
git clone https://github.com/ns1/waitron.git && cd waitron
docker build -t waitron . && docker-compose -f ./docker-compose.yml up
```

Then put your machine into build mode:

```
curl -X PUT http://my_waitron_location/build/dns02.example.com
```

If you've configured IPMI, Waitron and the example files will handle putting your new machine into PXE-mode on next boot, and it will handle power cycling your target machine.

If you _haven't_ configued IPMI, you'll need to power cycle the target machine to kick off the boot process, and you might also have to use the BIOS or a start-up key to force it to PXE boot.

From that point, Waitron should be able to handle the rest of the install process, and you can check status periodically if you like:

```
curl -X GET http://my_waitron_location/status/dns02.example.com
```

