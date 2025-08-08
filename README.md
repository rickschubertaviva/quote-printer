# Quote Printer

CLI tool to print a quote or policy to the terminal. Quicker than having to log into the AWS console and query DynamoDB!

# How To

- See help for options:

```sh
go run ./... -h
```

- Get a quote:

```sh
aws-vault-exec aviva-testing -- go run ./... dbf671ec-8412-4d52-ad58-4e65c4700dbc
```

- Get a policy (choose which state to inspect):

```sh
aws-vault exec aviva-testing -- go run ./... -policy dbf671ec-8412-4d52-ad58-4e65c4700dbc
```

- Automatically inspect the latest policy state ("STATE"):

```sh
aws-vault exec aviva-testing -- go run ./... -policy -latest dbf671ec-8412-4d52-ad58-4e65c4700dbc
```

# Helper scripts

There are helper bash scripts which you can use to define aliases in your zshrc files. That way, you can for example just type `qt` to see a quote in testing.

```sh
# Follow by quote ID to see it in testing
alias qt="aws-vault exec aviva-testing -- sh ~/code/quote-printer/quote.sh"
# Follow by quote ID to see it in staging
alias qt="aws-vault exec aviva-staging -- sh ~/code/quote-printer/quote.sh"
# Follow by policy ID to see the latest policy state in testing
alias pt="aws-vault exec aviva-testing -- sh ~/code/quote-printer/latest_policy.sh"
# Follow by policy ID to see the latest policy state in staging
alias ps="aws-vault exec aviva-staging -- sh ~/code/quote-printer/latest_policy.sh"
# Follow by policy ID to inspect a selected policy state in testing
alias pst="aws-vault exec aviva-testing -- sh ~/code/quote-printer/select_policy.sh"
# Follow by policy ID to inspect a selected policy state in staging
alias pss="aws-vault exec aviva-staging -- sh ~/code/quote-printer/select_policy.sh"
```
