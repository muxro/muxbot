# MuxBot Commands

* `.issues`
    - `issues` is a set of commands that revolve around gitlab issues in projects the bot is in (from the associated gitlab key entered when running). It is still a work in progress command and everything about it is subject to change. `activeRepo` changing is finished, but it isn't integrated in the `list` and `add` commands for now.
    - `.issues list` lists issues based on the parameters
        - Params:
            - `^author`: sets the required author of the issue.
            - `$assignee`: issues need to have been assigned to this assignee name.
                - exception: `$self` sets the assignee to the user if they have an associated gitlab key. If they don't, the command returns an error. 
            - `+tag`: adds tags that must be associated to the issue.
            - `&project`: issues need to be in this project.
    - `.issues add` adds an issue with the title being the text coming after it.
        - Usage: `.issues add <issue name> <params> <issue description>`
        - Params:
            - `$assignee`: issue is assigned to user.
            - `+tag`: issues will have this tag.
            - `&project`: issue is added to this project. If not set, it will be set to the active repository.
    
    - `.issues close` closes a specified issue and returns an error if it couldn't close it
        - Usage `.issues close <issue id>`
        - Params:
            - `issue id`: it's in the form `repo#id`, but if you wish to use your active repo you can leave the `repo#` part out

    - `.issues modify` updates an issue
        - Usage `.issues modify <issue id> <params>`
        - Params:
            - `<issue id>`: it's in the form `repo#id`, but if you wish to use your active repo you can leave the `repo#` part out
            - `$assignee`: updates the assignee
            - `+tag`: adds the specified tag
            - `-tag`: removes the specified tag (if it doesn't exist it isn't a problem)

    - `.issues active-repo` is a command used for setting the repository that the channel is working on.
        - Usage: `.issues active-repo <get/set/erase>`
        - `.issues active-repo get` displays the active repository for the channel in which the command was requested
        - `.issues active-repo set <repo>` sets the active repository for the requesting channel
        - `.issues active-repo erase` removes the active repository for the requesting channel