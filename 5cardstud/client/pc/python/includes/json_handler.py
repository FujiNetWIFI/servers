import json
import requests

class json_handler:
    
    def __init__(self,url):
        self.url = url
        self.json_data = None
        self.refresh_data()
        return
    
    
    def refresh_data(self):
        #response = requests.get(self.url)
        text = """
            {
                "num_of_players": 8,
                "round" : 4,
                "active_player": 1,

                "validMoves": [
                  {"move":"H",
                   "name" :"Check"},
                  {"move":"F",
                   "name" :"Fold"},
                  {"move":"C",
                   "name" :"Call"},
                  {"move":"RL",
                   "name" :"Raise 5"}, 
                  {"move":"RH",
                   "name" :"Raise 10"}, 
                  {"move":"BL",
                   "name" :"Bet 5"},
                  {"move":"BH",
                   "name" :"Bet 10"}
                ],
                
                "player":
                  [
                      {
                      "name"   : "Thom",
                      "bet"    : 105,
                      "purse"  : 95,
                      "hand"   : "ADKDAS8C",
                      "fold"   : 0
                      },
                      {
                      "name"   : "Norman",
                      "bet"    : 50,
                      "purse"  : 195,
                      "hand"   : "??8DJH",
                      "fold"   : 0
                      
                      },
                      {
                      "name"   : "Eric",
                      "bet"    : 50,
                      "purse"  : 15,
                      "hand"   : "??4H2S4H",
                      "fold"   : 0
                      },
                      {
                      "name"   : "Roger",
                      "bet"    : 50,
                      "purse"  : 15,
                      "hand"   : "??9C6S2H",
                      "fold"   : 0
                      },
                      {
                      "name"   : "mozzwald",
                      "bet"    : 50,
                      "purse"  : 95,
                      "hand"   : "??KDAS",
                      "fold"   : 0
                      },
                      {
                      "name"   : "Andy",
                      "bet"    : 50,
                      "purse"  : 195,
                      "hand"   : "??8DJH8C",
                      "fold"   : 0
                      },
                      {
                      "name"   : "Scoth42",
                      "bet"    : 50,
                      "purse"  : 15,
                      "hand"   : "??3H2S4H",
                      "fold"   : 0
                      },
                      {
                      "name"   : "eagle",
                      "bet"    : 50,
                      "purse"  : 15,
                      "hand"   : "??9C6S",
                      "fold"   : 1
                      }
                      
                    
                  ]
              }"""
        

        #self.json_data = json.loads(response.text);
        
        self.json_data = json.loads(text)
        json_data = json.loads(text)
        return True
    
    def get_number_of_players(self):
        return self.json_data['num_of_players']
    
    def get_name(self,player_num):
        return self.json_data["player"][player_num]["name"]
        
    def get_hand(self,player_num):
        return self.json_data['player'][player_num]['hand']
    
    def get_purse(self,player_num):
        return self.json_data['player'][player_num]['purse']
    
    def get_bet(self,player_num):
        return self.json_data['player'][player_num]['bet']
    
    def get_playing(self, player_num):
        player = self.json_data['active_player']
        player -= 1
        return player
    
    def get_fold(self, player_num):
        return self.json_data['player'][player_num]['fold']
    
    def get_round(self):
        return self.json_data['round']
    
    def get_valid_buttons(self):
        valid_moves = None
        
        moves = {}
        i = 0
        no_error = True
        while no_error:
            try:
                move = self.json_data['validMoves'][i]['move']
                name = self.json_data['validMoves'][i]['name']
                moves[move] = name
                i += 1
            except:
                no_error=False     
        
        return moves
    
    
