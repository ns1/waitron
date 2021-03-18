# waitron
> This project is in [maintenance](https://github.com/ns1/community/blob/master/project_status/MAINTENANCE.md) status.

[![Build Status](https://travis-ci.org/ns1/waitron.svg?branch=master)](https://travis-ci.org/ns1/waitron)

Waitron reads the machine definition from YAML and templates preseed and finish scripts based on that data. When a server is set in _build mode_ waitron will deliver a kernel/initrd/commandline used by [pixiecore](https://github.com/danderson/pixiecore) (in API mode) to boot and install the machine.

Run in docker container

    docker run -v /path/to/data:/data \
        -e CONFIG_FILE=/data/config.yaml \
        jhaals/waitron

Run locally

    go build . && CONFIG_FILE=config.yaml ./waitron

### config file
The config file needs a minimum set of parameters which will be available in the templates as **config._value_**.

name | description
--- | ---
templatepath | path where the _jinja2_ preseed, finish templates are located
machinepath | path where the _yaml_ machine definitions are located
baseurl | the url where this waitron instance will be listening

Extra parameters can be added in i.e. a params dictionari, those will be accessible in the templates as well

name | description
--- | ---
params.dns_servers | string containing the dns servers to be configured in the installed machines

### API

See [API.md](API.md) file in the repo

Contributions
---
Pull Requests and issues are welcome. See the [NS1 Contribution Guidelines](https://github.com/ns1/community) for more information.
