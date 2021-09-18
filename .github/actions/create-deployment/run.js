module.exports = async ({ github, context, core }) => {
  const applicationName = core.getInput('app-name', { required: true })

  const { data: deployment } = await github.repos.createDeployment({
    owner: context.repo.owner,
    repo: context.repo.repo,
    ref: context.payload.pull_request.head.ref,
    environment: `pr-${context.issue.number}-${applicationName}`,
    required_contexts: [],
    auto_merge: false,
  })

  core.setOutput('url', deployment.url)
}
