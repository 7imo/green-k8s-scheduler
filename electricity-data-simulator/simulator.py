from kubernetes import client, config


def main():
    
    try:
        config.load_incluster_config()
    except config.ConfigException:
        try:
            config.load_kube_config()
        except config.ConfigException:
            raise Exception("Could not configure kubernetes python client")

    k8s_api = client.CoreV1Api()

    body = {
        "metadata": {
            "labels": {
                "green": "true"}
        }
    }

    # Listing the cluster nodes
    node_list = k8s_api.list_node()

    print("%s\t\t%s" % ("NAME", "LABELS"))
    # Patching the node labels
    for node in node_list.items:
        api_response = k8s_api.patch_node(node.metadata.name, body)
        print(api_response)
        print("%s\t%s" % (node.metadata.name, node.metadata.labels))


if __name__ == '__main__':
    main()


#https://stackoverflow.com/questions/59741353/cannot-patch-kubernetes-node-using-python-kubernetes-client-library