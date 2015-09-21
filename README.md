# little-foreman
little-foreman read the machine definition from YAML and templates preseed and finish scripts based on that data. little-foreman contacts foreman-proxy's tftp API to set a machine in build mode(templates PXE configuration)

Run in docker container

    docker run -v /path/to/data:/data \
        -e CONFIG_FILE=/data/config.yaml
        jhaals/little-foreman

Run locally

    export CONFIG_FILE=config.yaml && go build . && ./templetation
