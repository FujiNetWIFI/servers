import os
import sys
from flask import Flask
# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)

from cards import *

# Create a Flask Instance
app = Flask(__name__)

global json_data

json_data = { }
global cards
cards = Cards(1)

global pool
pool = cards.create_card_pool(1)


def get_cards(num):
    hand = ""
    for i in range(num):
        hand += cards.get_card()
    return hand

def update_table():
    global json_data
    
    json_data["num_of_players"] = 8
    json_data["round"] = 4
    json_data["active_player"] = 1
    
    
def update_players():
    global json_data
    
    json_data["player"]=[]
    
    player= {}
    player["name"] = "Thom"
    player['bet']  = 105
    player['purse']= 95
    player['hand'] = get_cards(4)
    player['fold'] = 0
    json_data["player"].append(player)
    
    player= {}
    player["name"]   = "Norman"
    player["bet"]    = 50
    player["purse"]  = 195
    player["hand"]   = "??" + get_cards(3)
    player["fold"]   = 0
    json_data["player"].append(player)
    
    player= {}
    player["name"]   = "Eric"
    player["bet"]    = 50
    player["purse"]  = 15
    player["hand"]   = "??" + get_cards(3)
    player["fold"]   = 0
    json_data["player"].append(player)
    
    player= {}
    player["name"]   = "Roger"
    player["bet"]    = 50
    player["purse"]  = 15
    player["hand"]   = "??" + get_cards(3)
    player["fold"]   = 0
    json_data["player"].append(player)
    
    player= {}     
    player["name"]   = "mozzwald"
    player["bet"]    = 50
    player["purse"]  = 95
    player["hand"]   = "??" + get_cards(3)
    player["fold"]   = 0
    json_data["player"].append(player)
    
    player= {}     
    player["name"]   = "Andy"
    player["bet"]    = 50
    player["purse"]  = 195
    player["hand"]   = "??" + get_cards(3)
    player["fold"]   = 0
    json_data["player"].append(player)
    
    player= {}     
    player["name"]   = "Scoth42"
    player["bet"]    = 50
    player["purse"]  = 15
    player["hand"]   = "??" + get_cards(3)
    player["fold"]   = 0
    json_data["player"].append(player)
    
    player= {}     
    player["name"]   = "eagle"
    player["bet"]    = 50
    player["purse"]  = 15
    player["hand"]   = "??9C6S"
    player["fold"]   = 1
    json_data["player"].append(player)
    
    
    
def update_moves():
    global json_data
    
    json_data["validMoves"]=[]
    
    move = {}
    move['move'] = "H"
    move['name'] = "Check"
    json_data["validMoves"].append(move)
    
    move = {}
    move['move'] = "F"
    move['name'] = "Fold"
    json_data["validMoves"].append(move)
    
    move = {}
    move['move'] = "C"
    move['name'] = "Call"
    json_data["validMoves"].append(move)
    
    move = {}
    move['move'] = "RL"
    move['name'] = "Raise 5"
    json_data["validMoves"].append(move)
    
    move = {}
    move['move'] = "RH"
    move['name'] = "Raise 10"
    json_data["validMoves"].append(move)
    
    move = {}
    move['move'] = "BL"
    move['name'] = "Bet 5"
    json_data["validMoves"].append(move)
    
    move = {}
    move['move'] = "BH"
    move['name'] = "Bet 10"
    json_data["validMoves"].append(move)
    
    move = {}
    

# Json Thing
@app.route('/5cardstud')
def get_5card_stud_state():
    update_table()
    update_players()
    update_moves()
    return json_data

json_data = {}

json_data["player"]=[]


