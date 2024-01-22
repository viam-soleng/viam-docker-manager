# viam-docker-manager

[![Go](https://github.com/viam-soleng/viam-docker-manager/actions/workflows/go.yml/badge.svg)](https://github.com/viam-soleng/viam-docker-manager/actions/workflows/go.yml)

## Description

This is a module for Viam Robotics to manage docker containers, on your robot, with the RDK. You can find this module in the [Viam Registry](https://app.viam.com/module/viam-soleng/viam-docker-manager)

## Usage

1. [Configure a new Component](https://docs.viam.com/registry/configure/) in your robot using [app.viam.com](app.viam.com)
2. Search for "docker" and click the "sensor/manage:docker" from "viam-soleng"
3. Click "Add module"
4. Name the component (ex: `container0`)
5. Click "Create"
6. Create the relevant attributes (see [config](#config))


## Config

You can find the entire config in [config.go](docker/config.go#L11-L20).

This module can start containers in one of two ways (per-component), using `docker run` or using `docker compose ... up`

### Root Config
|Attribute|Required|Type|Description|
|---------|--------|----|-----------|
|run_options|N|RunOptions|Options for starting a container with the equivalent of `docker run`|
|compose_options|N|ComposeOptions|Options for starting a container (or containers) with the equivalent of `docker compose`|
|image_name|Y|string|The name of the image on Docker Hub or the full name of the image and registry if not using Docker Hub|
|repo_digest|Y|string|The digest hash of the image on the repository|
|run_once|N|bool|Only run the container once|
|download_only|N|bool|Only download the container, don't attempt to start it|
|credentials|N|Credentials|Credentials to use for pulling images from a private repository|

### RunOptions
|Attribute|Required|Type|Description|
|---------|--------|----|-----------|
|entry_point_args|N|[]string|The command to pass as the entrypoint to the container|
|options|N|[]string|Any options to also pass to the container|

### ComposeOptions
|Attribute|Required|Type|Description|
|---------|--------|----|-----------|
|compose_file|Y|[]string|The contents of the docker compose file, each line of the file is a single entry in the array, whitespace is preserved|

_Note: The image tag in the `compose_file` is **required** and **must** match the `image_name` and `repo_digest` provided in the attributes._

---

## Usage

### `docker run`

Given the following `docker run` command examples, here is how they are translated to a component configuration for this module.

#### Basic Example

Command: `docker run ubuntu echo hi`
Attributes:
```
{
    "image_name": "ubuntu",
    "repo_digest": "sha256:04714a1bfbb2d8b5390b5cc0c055e48ebfabd4aa395821b860730ff3277ed74a",
    "run_options": {
        "entry_point_args": [
            "echo",
            "hi"
        ]
    }
}
```

#### Basic Example with Options

Command: `docker run --rm ubuntu echo hi`
Attributes:
```
{
    "image_name": "ubuntu",
    "repo_digest": "sha256:04714a1bfbb2d8b5390b5cc0c055e48ebfabd4aa395821b860730ff3277ed74a",
    "run_options": {
        "entry_point_args": [
            "echo",
            "hi"
        ],
        options: [
            "--rm"
        ]
    }
}
```
---

### `docker compose`

#### Sample Config

```
{
    "image_name": "ubuntu",
    "repo_digest": "sha256:04714a1bfbb2d8b5390b5cc0c055e48ebfabd4aa395821b860730ff3277ed74a",
    "compose_options": {
        "compose_file": [
            "services:",
            "    app:",
            "      image:ubuntu@sha256:04714a1bfbb2d8b5390b5cc0c055e48ebfabd4aa395821b860730ff3277ed74a",
            "      command: echo hi"
            "      working_dir: /root"
        ]
    }
}
```

## FAQ
* Why does the `image` tag in the compose file have to match the `image_name` and `repo_digest` provided in the config?
   * If they don't, starting the compose file may fail, or cause an unexpected delay in robot startup while the required images are downloaded.
* Does this mean I can't use the compose option to start multiple containers?
   * No, in theory it should work as long as one of the `image` tags matches the `image_name` and `repo_digest`. Given the problem listed above, I discourage using compose files.
