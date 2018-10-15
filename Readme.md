
# Klippy

## A tool for helping deploy container-based applications

```
 __                 
/  \        _______________ 
|  |       /               \
@  @       | It looks      |
|| ||      | like you      |
|| ||   <--| are deploying |
|\_/|      | a container.  |
\___/      \_______________/
```



### Images

Image support is the only function available at the moment, and allows `klippy` to interact with any registry that supports the `Version 2` API (details here https://docs.docker.com/registry/spec/api/#detail).

#### List tags of an image

The command `klippy image tags --name <image>` query a registry and retrieve all tags for a particular image. If no Registry hostname is specified then `klippy` will default the hub.docker.com mimicking the same behaviour as the docker cli.

#### List commands used to build an image

The command `klippy image commands --name <image>` will again query a registry and retrieve all of the commands that were used to build the entire image. 

**NOTE** The lines in red are `NOP` lines, as in no actual commands are run within the layer.