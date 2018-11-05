# mobhealth3-prefill

**mobhealth3-prefill** extracts mob health point information from a mangos database dump (aka WoW vanilla DB) and exports it in the format understood by [MobHealth3](https://wow.curseforge.com/projects/project-2615). The dump can be extracted from a 7zip archive on the fly.

## Usage

``` shell
go build
git clone https://github.com/brotalnia/database.git
./mobhealth3-prefill database/world_full_26_august_2018.7z
```

Then copy `MobHealth.lua` to your account wide `SavedVariables` directory. You might want to preserve the configuration part of your previous file.