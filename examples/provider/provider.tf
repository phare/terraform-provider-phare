terraform {
  required_providers {
    phare = {
      source  = "phare/phare"
      version = "~> 0.0.1"
    }
  }
}

# Set this variable in a *.tfvars file
# or define a PHARE_API_KEY environment variable instead
variable "phare_api_key" {
  sensitive = true
}

# Configure the Phare Provider
provider "phare" {
  api_key = "your-api-key-here"

  # Optional: Override the default API base URL
  base_url = "https://api.phare.io"

  # Optional: HTTP client timeout in seconds (default: 30)
  timeout = 30

  # Optional: When using an organization-scoped API key you can set
  # a global project scope that will be used on all resources
  project_scope = "your-project-slug"
}
