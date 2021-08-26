from kubernetes import client, config
import random
import time


def main():
    
    try:
        config.load_incluster_config()
    except config.ConfigException:
        try:
            config.load_kube_config()
        except config.ConfigException:
            raise Exception("Could not configure kubernetes python client")

    k8s_api = client.CoreV1Api()

    while True:
        annotate_nodes(k8s_api)
        time.sleep(120)


def annotate_nodes(k8s_api):
     # Listing the cluster nodes
    node_list = k8s_api.list_node()

    # Patching the node annotations
    for node in node_list.items:
        renewable_share_float = random.uniform(0, 1)
        print("Node %s has a renewable energy share of %s" % (node.metadata.name, renewable_share_float))
        renewable_share = "{:.1f}".format(renewable_share_float)
        #TODO: Round down to two last digits
        print("The renwable share will be %s" % renewable_share)
        body = {
                "metadata": {
                    "annotations": {
                        "renewable": renewable_share }
                }
            }

        api_response = k8s_api.patch_node(node.metadata.name, body)
        # print(api_response)
        

if __name__ == '__main__':
    main()
