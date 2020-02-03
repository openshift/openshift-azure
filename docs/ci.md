# CI

## Application rotation "need to know"

Key points to know when re-creating AAD applications (CI or Shared teams):

1. Application should have requires API permission assigned to them. Check
before deleting old applications.
1. All teams and CI shared application should have corresponding AAD application
with "owners" permissions. Check "Owners" tab on the application.
This can be achieved using command: `./hack/aad.sh app-create aro-ci-aad-shared aro-ci-team-shared`
1. All shared application should have `User Access Administrator` permissions
assigned to it. This is not shown in the web UI. It can be achieved with command:
`az role assignment create --assignee "f7c45571-4c87-431d-a8e3-90f87d1a4fe4" --role "User Access Administrator"`
where `assignee` is shared application ID
