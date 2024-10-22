## Next steps

- [ ] go over the rules to match directories, because currently it uses Contains, which is a bit too broad. We should use something like gitignore with / at the end or not. This links to the task of fixing gitignore package.

## Glazed converstion

- [x] convert to glazed command
- [x] restore short flags
- [x] restore CATTER_PROFILE env variable
- [x] print out stats using glazed
- [x] turn filter into a utility package
- [x] add glazed flag layer for filter options (to reuse in other commands)
- [x] fix print-filters
- [x] fix catter profile

- [ ] make a custom verb for the stats
- [ ] add glazed help system for catter

## YAML settings

- [x] Add profiles to catter.yaml
- [x] allow loading of filter rules from a .catter.yaml file
- [x] load profile from CATTER_PROFILE env variable

## Ideas

- [x] Filter out binary files
- [ ] Verify gitignore to ignore .history for example
- [ ] Add web API + rest 
- [ ] add a filter to process each file (maybe?? with lua or bash commands? Look at how git hooks are defined?)

## Done

- [x] Add more succinct flags
- [x] fix bug when computing stats
- [x] Show token count per directory / file type
- [x] AST walker generic per file / per directory (when called with directory, given list of files in the directory, potentially also the results for each of the files in that directory first).
- [x] properly count files when computing stats
