terraform {
  required_version = ">= 1.6.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
  }
}

variable "kubeconfig" {
  type        = string
  description = "Path to the target cluster kubeconfig."
}

provider "kubernetes" {
  config_path = var.kubeconfig
}

resource "kubernetes_namespace" "platform" {
  metadata { name = "webhook-platform" }
}

resource "kubernetes_config_map" "service_contract" {
  metadata { name = "webhook-service-contract" namespace = kubernetes_namespace.platform.metadata[0].name }
  data = {
    health_endpoint   = "/healthz"
    metrics_endpoint  = "/metrics"
    delivery_contract = "POST /v1/events with X-Tenant-ID and idempotent event_id"
  }
}
