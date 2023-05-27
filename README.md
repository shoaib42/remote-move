# remote-move

Provides a web interface to facilitate moving files on the server, with chown-ing the files to the desired user.

***

## Why not do it via terminal

Go for it! I'd rather be on my couch and move transferred files (absolutely not from transmission) to my media folders (absolutely not to jellyfin folders) using a web interface on my phone, and proceede to hit play (absolutely legal stuff).

***

## How to use it?

Go through the configurations in [configuration.yaml](configuration.yaml)
```
go build
sudo remote-move
```
Yes it must be run as root.

DIY setup a service,

Modify and use as you like.

***

## Was chatGPT used?

Shhhh! Only for the front end, I'm not a front end guy. chatGPT sucks with pretty much everything else.

***

## Look and feel
***
### Web
![Web](/img/web.png)
***
Select multiple

![Select multiple](/img/move_selected.png)

***

## Moves and Chowns

### Before

![before](/img/before.png)

***

### After

![after](/img/aftermove.png)

