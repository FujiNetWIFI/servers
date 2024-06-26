import os
import sys
import pygame

# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)
    
from common import *

def send_action_to_server(action):
    common_vars   = CommonVariables.get_instance()
    common_vars.server.send_action(action[5])
    if not common_vars.server.connected:
        print(f"{url} is down.")
        common_vars.done = True



    