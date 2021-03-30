### Historical info and credits
The original Waitron, found on the version 1.0.0 branch in this repo, was originally written by [jhaals](https://github.com/jhaals).
We at NS1 needed an internal build system that would allow us to meet a specific set of requirements and found [pixiecore](https://github.com/danderson/pixiecore), and eventually Waitron in an unmaintained state.  Jhaals was kind enough to let NS1 take over the project, and we've continued maintaining it since.

The 2.0.0 branch of this repo still has a large portion of the original from Jhaals, with a few additions we needed at the time.  However, the current main branch (representing post 2.0.0) is an almost complete rewrite of the original Waitron code.


# Waitron
> This project is in [maintenance](https://github.com/ns1/community/blob/master/project_status/MAINTENANCE.md) status.

[![Build Status](https://travis-ci.org/ns1/waitron.svg?branch=master)](https://travis-ci.org/ns1/waitron)

Waitron is used to build machines (primarily bare-metal, but anything that understands PXE booting will work) based on definitions from any number of specified inventory sources.

When a server is set in _build mode_, Waitron will deliver a kernel/initrd/commandline that can be used by [pixiecore](https://github.com/danderson/pixiecore) (in API mode) to boot and install the machine.

Try it out in a docker:

```
docker build -t waitron . && docker-compose -f ./docker-compose.yml up
```

```
$ curl -X PUT http://localhost:7078/build/dns02.example.com
{"Token":"fb300739-b4ce-4740-af26-80a99326ee05"

$ curl -X GET http://localhost:7078/status/dns02.example.com
pending

curl -X PUT http://localhost:7078/cancel/dns02.example.com/fb300739-b4ce-4740-af26-80a99326ee05
{"State":"OK"}

```

### Config file
See the example [config](examples/config.yml) for descriptions and examples of configuration options.

### API

See [API.md](API.md) file in the repo

Contributions
---
Pull Requests and issues are welcome. See the [NS1 Contribution Guidelines](https://github.com/ns1/community) for more information.
