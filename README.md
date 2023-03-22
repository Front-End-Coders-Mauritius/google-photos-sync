# Sync Photos from Front-end Coders Google Albums

The repo is the storage for our event albums. 
It is meant to be consumed by [Front-End-Coders-Mauritius/meetup-fec-nuxt](https://github.com/Front-End-Coders-Mauritius/meetup-fec-nuxt) for frontend.mu

The script is run locally and maintained by [sandeep@ramgolam.com](https://github.com/MrSunshyne)

# MAKE SURE YOU HAVE `timerliner.toml` in the root directory. 

It should contain: 

```
[oauth2.providers.google]
client_id = "xxx"
client_secret = "xxx"
auth_url = "https://accounts.google.com/o/oauth2/auth"
token_url = "https://accounts.google.com/o/oauth2/token"
```

## Commands 

`make init` get all photos from all albums in the fec google account

`make sync` get latest photos

`make json` run the go utility by [@mgjules](https://github.com/mgjules) to create an index.json of all the photos grouped by Album Name
	
## What should I do to sync photos? 

```bash
make sync
make json 
```
