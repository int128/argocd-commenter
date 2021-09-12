# end-to-end test

To run locally:

```sh
# set up a cluster and deploy argocd
make deploy-argocd

# set up main branches
make setup

# run the controller

# create pull requests
make test

# clean up branches
make cleanup
```
