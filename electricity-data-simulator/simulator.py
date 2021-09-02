from kubernetes import client, config
import random
import time

SCHEDULING_RENEWABLE_THRESHOLD = 0.3

# initialize k8s config
try:
    config.load_incluster_config()
except config.ConfigException:
        try:
            config.load_kube_config()
        except config.ConfigException:
            raise Exception("Could not configure kubernetes python client")

k8s_api = client.CoreV1Api()

# initialize node_shares dict for taints
node_shares = {}


def main():
    
    # get all nodes in the cluster
    node_list = k8s_api.list_node()

    while True:
        # update annotations with recent energy data
        for node in node_list.items:
            annotate_node(node.metadata.name)

        # set node taints for de/rescheduling of pods
        taint_nodes()

        # repeat after 5 minutes
        time.sleep(120)


def annotate_node(node_name):
        
    # get energy data for node
    renewable_share_float = random.uniform(0, 1)

    # formatting
    renewable_share = "{:.1f}".format(renewable_share_float)

    print("Node %s has a renewable energy share of %s" % (node_name, renewable_share))

    # annotation body
    annotations = {
                "metadata": {
                    "annotations": {
                        "renewable": renewable_share }
                }
            }

    # send to k8s
    response = k8s_api.patch_node(node_name, annotations)
    # print(response)

    # track current values for nodes
    node_shares[node_name] = renewable_share    
        

def taint_nodes():

    for key, value in node_shares.items():

        if float(value) <= SCHEDULING_RENEWABLE_THRESHOLD: 
            # activates NoExecute Taint Policy for Node, which causes Pods to be descheduled
            k8s_api.patch_node(key, {"spec":{"taints":[{"effect":"NoExecute", "key":"green", "value":"false"}]}})
            print("Descheduling Pods from Node %s, which has a renewable energy share of %s" % (key, value))
        else:
            # deletes any Taint Policy, Pods can be scheduled to Node again
            k8s_api.patch_node(key, {"spec":{"taints":[]}})
            print("Allowing Pods on Node %s, which has a renewable energy share of %s" % (key, value))
        

if __name__ == '__main__':
    main()

