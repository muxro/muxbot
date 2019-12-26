# MuxBot Commands

* `.help`
    - `help` replies to people with a link to this file, providing documentation for the bot's available commands.

* `.ping`
    - `ping` replies to the user with `pong`, to test the latency between the user and the bot.

* `.echo`
    - `echo` replies back with the text sent by the user. There aren't many use cases for it but it's a nice-to-have.
    - Usage: `.echo <text>`

* `.eval`
    - `eval` uses [govaluate](https://github.com/Knetic/govaluate) for evaluating simple mathematical expressions.
    - Usage: `.eval <expression>`

* `.g`
    - `g` scrapes the first web result on dogpile.com (a search engine based on bing) for the desired query with the link and the description of the result. Even though it's not google, it returns on-topic results.
    - Usage: `.g <query>`

* `.gis`
    - `gis` acts like `g`, but instead of scraping the first web result, it scrapes the first image result
    - Usage: `.gis <query>`

* `.yt`
    - `yt` queries the Youtube API and replies with a link to the first youtube video result
    - Usage: `.yt <query>`

* `.issues`
    - `issues` is a set of commands that revolve around gitlab issues in projects the bot is in (from the associated gitlab key entered when running). It is still a work in progress command and everything about it is subject to change. `activeRepo` changing is finished, but it isn't integrated in the `list` and `add` commands for now.
    - Usage: `.issues <list/add/activeRepo>`
    - `.issues list` lists issues based on the parameters
        - Params:
            - `^author`: sets the required author of the issue.
            - `$assignee`: issues need to have been assigned to this assignee name.
                - exception: `$self` sets the assignee to the user if they have an associated gitlab key. If they don't, the command returns an error. 
            - `+tag`: adds tags that must be associated to the issue.
            - anything else is treated as *the* repositories to search in. If you have multiple repos listed, it searches the last one entered.
    - `.issues add` adds an issue with the title being the text coming after it. It is planned to support assignees and tags with `issues list`-like syntax, but we haven't decided on how descriptions should be handled.
    - `.issues activeRepo` is a command used for setting the repository that the user is working on.
        - Usage: `.issues activeRepo <get/set/erase>`
        - `.issues activeRepo get` displays the active repository for the user that requested the command
        - `.issues activeRepo set <repo>` sets the active repository for the requesting user
        - `.issues activeRepo erase` removes the active repository for the requesting user

* `.glkey`
    - `glkey` associates a discord user with a personal access token and is used by `.issues` when `list`ing issues assigned to `$self` and `add`ing issues
    - Usage: `.glkey <personal access token>`

* `.todo`
    - `todo` is an unfinished command and should not be used for now. It will be documented more thoroughly when it is finished.
    - Objectives: `.todo <add/remove/clean/move/rename/done>`
    - Completed: `.todo <add/create/done/clean>`