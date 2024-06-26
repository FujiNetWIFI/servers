import os
import sys
import pygame

# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)
    
from common import *
from button_handler import *

def event_handler():
    
    for event in pygame.event.get():
        if event.type == pygame.QUIT:
            common_vars      = CommonVariables.get_instance()
            common_vars.done = True
            
        if event.type == pygame.MOUSEBUTTONUP and event.button == 1:
            button_handler()
            

