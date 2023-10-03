# Release Process

## Resolve Issues
1. Create [a new milestone](https://github.com/aregm/enzyme/milestones) if one doesn't exist.
1. Open [issues](https://github.com/aregm/enzyme/issues) and filter by milestone and make sure they are either closed or moved over to the next milestone.

## Start a release PR
1. Run [Generate Enzyme Manifests workflow](https://github.com/aregm/enzyme/workflows/generate_manifest.yml). It’ll create a PR ([example](https://github.com/aregm/enzyme/pull/0000))
1. Update docs to match the milestone version.
1. Create a CHANGELOG file ([example]())
1. Wait for end-to-end tests to finish then merge PR.

## Create a release
1. Run [Create Enzyme Release workflow](https://github.com/aregm/enzyme/workflows/create_release.yml):
   It will create a tag and then publish all deployment manifest in github release and will create a discussion thread in github release
1. Kick off a run of the functional tests. 
1. Close the milestone
1. Ping #core ([Discord]() channel) to send announcements about the milestone with the contents of the CHANGELOG to all social channels.