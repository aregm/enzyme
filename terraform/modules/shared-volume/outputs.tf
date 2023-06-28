output "namespace" {
  value = kubernetes_namespace.shared-volume.id
}

output "config_map" {
  value = kubernetes_config_map.shared-volume.metadata.0.name
}
