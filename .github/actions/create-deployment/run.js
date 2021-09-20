module.exports = async ({github, context, core}, appName) => {
  let ref
  let environment
  if (context.eventName === 'pull_request') {
    ref = context.payload.pull_request.head.ref
    environment = `pr-${context.issue.number}-${appName}`
  } else {
    ref = context.ref
    environment = `${ref}-${appName}`
  }

  const {data: deployment} = await github.repos.createDeployment({
    owner: context.repo.owner,
    repo: context.repo.repo,
    ref,
    environment,
    required_contexts: [],
    auto_merge: false,
  })

  core.setOutput('url', deployment.url)
}
