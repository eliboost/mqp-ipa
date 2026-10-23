# Version Sync (VerSync)
Versync is an automation tool designed to review and synchronize versions of middleware activemq components (such as Artemis, Artemis cloud operator and PF operator). It scans defined GitLab repositories, analyzes `kustomization.yaml` files, and extracts version information from operators and images to ensure that documentation is up to date.

## Project Structure

- **/activemqpf:** Contains logic for extracting the **Artemis Version** and the **PF ActiveMQ Operator Version**.
- **/artemis:** Contains logic for extracting the **Artemis Cloud Operator Version**.
- **/config:** Defines project IDs and other configuratin values for the project.

## Functions

- **GitLab Integration:** Connects to the GitLab API to browse repository trees and files.
- **Kustomize Parsing:** Reads and parses `kustomization.yaml` files to extract image tags.
- **Version Extraction:**
  - Artemis Version
  - Artemis Cloud Operator Version
  - PF ActiveMQ Operator Version

## Usage

Configure the required environment variables before running the tool:

`GITLAB_TOKEN`:: Your GitLab Personal Access Token with read/write permissions

`BRANCH_NAME`:: The target branch name for all ActiveMQ repositories (documentation, versync, kube-artemis-operator, and kube-pf-activemq-operator)