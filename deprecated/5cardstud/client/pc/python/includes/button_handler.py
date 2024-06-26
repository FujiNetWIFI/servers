import os
import sys
import pygame

# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)
    
from common import *
from button_handler import *


def erase_buttons():
    common_vars   = CommonVariables.get_instance()
    common_vars.buttons = []
    
    
def add_button(x1,y1,x2,y2,routine,key):
    common_vars   = CommonVariables.get_instance()
    common_vars.buttons.append( (x1,y1,x2,y2,routine,key))
    

def button_handler():
    common_vars   = CommonVariables.get_instance()
    mouse_x, mouse_y = pygame.mouse.get_pos()

    for button in common_vars.buttons:

        if mouse_x>=button[0] and mouse_x<=button[2]:
            if mouse_y >= button[1] and mouse_y <= button[3]:
                button[4](button)

                

    

