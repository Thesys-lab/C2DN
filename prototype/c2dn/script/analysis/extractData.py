

import os, sys
sys.path.append(os.path.expanduser("~/workspace/"))
from pyutils.common import * 


def load_fe_metrics(ifilepath):
    n_byte_partial_miss, n_req_partial_miss = 0, 0
    n_byte_push_chunk, n_byte_chunk_hit, n_req_chunk_hit, n_byte_ICP_chunk = 0, 0, 0, 0
    n_req_ICP_chunk, n_req_skip_chunk = 0, 0
    n_req_chunk_resp_skipped = 0

    with open(ifilepath) as ifile:
        for line in ifile:
            if not line.startswith("frontend"):
                continue 
            if 'byte{reqType="allToClient"}' in line:
                n_byte_to_client = float(line.split()[1])
            elif 'nReq{reqType="allToClient"}' in line:
                n_req_to_client = float(line.split()[1])

            elif 'trafficType="origin"' in line:
                n_byte_from_origin = float(line.split()[1])
            elif 'reqType="fullObjMiss"' in line:
                n_req_from_origin = float(line.split()[1])



            elif 'traffic{trafficType="intra"}' in line:
                n_byte_intra = float(line.split()[1])

            elif 'traffic{trafficType="ICPFull"}' in line:
                n_byte_ICP_full = float(line.split()[1])
            elif 'traffic{trafficType="ICPChunk"}' in line:
                n_byte_ICP_chunk = float(line.split()[1])

            elif 'trafficType="pushFullObj"' in line:
                n_byte_push_full = float(line.split()[1])
            elif 'trafficType="pushChunk"' in line:
                n_byte_push_chunk = float(line.split()[1])


            elif 'nReq{reqType="ICPFull"}' in line:
                n_req_ICP_full = float(line.split()[1])
            elif 'nReq{reqType="ICPChunk"}' in line:
                n_req_ICP_chunk = float(line.split()[1])
            elif 'nReq{reqType="skipFetch"}' in line:
                n_req_skip_chunk = float(line.split()[1])
            elif 'frontend_nReq{reqType="chunkRespSkipped"}' in line:
                n_req_chunk_resp_skipped = float(line.split()[1])


            # elif 'traffic{trafficType="pushChunk"}' in line:
            #     n_byte_push_chunk = float(line.split()[1])

            elif 'byte{reqType="chunkHit"}' in line:
                n_byte_chunk_hit = float(line.split()[1])
            elif 'nReq{reqType="chunkHit"}' in line:
                n_req_chunk_hit = float(line.split()[1])

            elif 'byte{reqType="partialHit_1"}' in line:
                n_byte_partial_miss += float(line.split()[1]) / 3 * 2
            elif 'byte{reqType="partialHit_2"}' in line:
                n_byte_partial_miss += float(line.split()[1]) / 3

            elif 'nReq{reqType="partialHit_1"}' in line:
                n_req_partial_miss += float(line.split()[1]) 
            elif 'nReq{reqType="partialHit_2"}' in line:
                n_req_partial_miss += float(line.split()[1]) 


    ret_dict = {
        "n_byte_to_client": n_byte_to_client, 
        "n_req_to_client": n_req_to_client, 
        "n_byte_from_origin": n_byte_from_origin, 
        "n_req_from_origin": n_req_from_origin, 
        "n_byte_intra": n_byte_intra, 
        "n_byte_ICP_full": n_byte_ICP_full, 
        "n_req_ICP_full": n_req_ICP_full, 
        "n_byte_push_full": n_byte_push_full, 
        "n_byte_push_chunk": n_byte_push_chunk, 

        "n_byte_chunk_hit": n_byte_chunk_hit, 
        "n_req_chunk_hit": n_req_chunk_hit, 

        "n_req_skip_chunk": n_req_skip_chunk, 
        "n_req_chunk_resp_skipped": n_req_chunk_resp_skipped, 

        "n_byte_ICP_chunk": n_byte_ICP_chunk, 
        "n_req_ICP_chunk": n_req_ICP_chunk, 
        "n_byte_partial_miss": n_byte_partial_miss, 
        "n_req_partial_miss": n_req_partial_miss, 
    } 
    return ret_dict

def load_all_fe_metrics(ifile_dir, system):
    all_data = []
    for i in range(10):
        try:
            d = load_fe_metrics("{}/cdn{}/c2dn/metricFE".format(ifile_dir, i))
            all_data.append(d)
        except Exception as e:
            print(e)

    client_bytes = sum([d["n_byte_to_client"] for d in all_data])
    origin_bytes = sum([d["n_byte_from_origin"] for d in all_data])
    client_nreq = sum([d["n_req_to_client"] for d  in all_data])
    origin_nreq = sum([d["n_req_from_origin"] for d in all_data])
    intra_bytes = sum([d["n_byte_intra"] for d in all_data])
    # this is not accurate as it includes skipped chunk fetch 
    intra_get_bytes = sum([d["n_byte_ICP_full"] for d in all_data])
    intra_push_bytes = sum([d["n_byte_push_full"] for d in all_data])

    intra_get_nreq = sum([d["n_req_ICP_full"] for d in all_data])

    if system == "C2DN":
        # intra_get_nreq += (sum([d["n_req_ICP_chunk"] for d in all_data]) - sum([d["n_req_skip_chunk"] for d in all_data]))//3
        # intra_get_nreq += sum([d["n_req_ICP_chunk"] for d in all_data]) // 3 
        intra_get_bytes += sum([d["n_byte_ICP_chunk"] for d in all_data])
        intra_push_bytes += sum([d["n_byte_push_chunk"] for d in all_data])

    print("bmr {:.4f} omr {:.4f} | bytes intra {:.4f} intra_get {:.4f} intra_push {:.4f} | nReq intra get (full) {:.4f}".format(
        origin_bytes/client_bytes, origin_nreq/client_nreq, 
        intra_bytes/client_bytes, intra_get_bytes/client_bytes, intra_push_bytes/client_bytes, 
        intra_get_nreq/client_nreq, 
        ))

    if system == "C2DN": 
        chunk_serve_nreq = sum([d["n_req_chunk_hit"] for d in all_data])
        chunk_serve_nreq += sum([d["n_req_partial_miss"] for d in all_data])
        chunk_serve_bytes = sum([d["n_byte_chunk_hit"] for d in all_data])
        chunk_serve_bytes += sum([d["n_byte_partial_miss"] for d in all_data])
        print("serving with chunks: {:.4f} req {:.4f} bytes".format(
            chunk_serve_nreq/client_nreq, chunk_serve_bytes/client_bytes, 
            ))




if __name__ == "__main__": 
    BASE_DIR = "/nvme/log/p/2021-02-01/"
    # load_all_fe_metrics(f"{BASE_DIR}/0124/aws_CDN_akamai2_expLatency_unavail0_1000G/", system="CDN") 
    # load_all_fe_metrics(f"{BASE_DIR}/0124/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/", system="C2DN") 

    # load_all_fe_metrics(f"{BASE_DIR}/0125/aws_CDN_akamai2_expLatency_unavail1_1000G/", system="CDN") 
    # load_all_fe_metrics(f"{BASE_DIR}/0125/aws_C2DN_akamai2_expLatency_unavail1_43_1000G/", system="C2DN") 

    # load_all_fe_metrics(f"{BASE_DIR}/0127/aws_CDN_akamai1_expLatency_unavail0_100G/", system="CDN") 
    # load_all_fe_metrics(f"{BASE_DIR}/0127/aws_C2DN_akamai1_expLatency_unavail0_43_100G/", system="C2DN") 


    # load_all_fe_metrics(f"{BASE_DIR}/0130/aws_CDN_akamai1_expLatency_unavail0_100G/", system="CDN") 
    # load_all_fe_metrics(f"{BASE_DIR}/0130/aws_C2DN_akamai1_expLatency_unavail0_43_100G/", system="C2DN") 

    load_all_fe_metrics(f"{BASE_DIR}/aws_CDN_akamai2_expLatency_unavail0_1000G/", system="CDN") 
    load_all_fe_metrics(f"{BASE_DIR}/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/", system="C2DN") 




