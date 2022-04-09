

#### frontend 
# small object local miss 
curl -svo /tmp/feResp -H "bucket: 1" http://127.0.0.1:8081/akamai/1_1024 && ls -l /tmp/feResp
# small object local hit 
curl -svo /tmp/feResp -H "bucket: 1" http://127.0.0.1:8081/akamai/1_1024 && ls -l /tmp/feResp
# small object ram hit 
curl -svo /tmp/feResp -H "bucket: 1" http://127.0.0.1:8081/akamai/1_1024 && ls -l /tmp/feResp


# large object local miss 
curl -svo /tmp/feResp -H "bucket: 1" http://127.0.0.1:8081/akamai/2_1096000 && ls -l /tmp/feResp
# large object local hit 
curl -svo /tmp/feResp -H "bucket: 1" http://127.0.0.1:8081/akamai/2_1096000 && ls -l /tmp/feResp






# test remote client and remote origin 







