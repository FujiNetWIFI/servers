import os
import sys
import pygame

# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)
    
from common import *

def check_button():
    print("check")

def call_button():
    print("call")
    
def fold_button():
    print("fold")
    
def play_button():
    print("play")

def raise_lower_button():

    common_vars   = CommonVariables.get_instance()  
    common_vars.player_bets.append(5)
    common_vars.new_chip_data = True

def raise_higher_button():
    common_vars   = CommonVariables.get_instance()
    common_vars.player_bets.append(10)
    common_vars.new_chip_data = True
    
def bet_lower_button():

    common_vars   = CommonVariables.get_instance()   
    common_vars.player_bets.append(5)
    common_vars.new_chip_data = True

def bet_higher_button():
    common_vars   = CommonVariables.get_instance()
    common_vars.player_bets.append(10)
    common_vars.new_chip_data = True

    

    