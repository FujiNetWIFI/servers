import json
import requests

class json_handler:
    
    def __init__(self,url):
        self.url = url
        self.json_data = None
        self.data_change = False
        self.last_data = "@"
        self.refresh_data()
        self.table = ""
        return
    
    def set_table(self, table):
        self.table = "?table="+table
        print(self.table)
        requests.get(self.url+"/state"+self.table)
        
        
        
    def set_players(self, players):
        print(self.url+"/state?count="+str(players)+self.table)
        requests.get(self.url+"/state?count="+str(players)+self.table)
        
    def refresh_data(self):
        try:
            response = requests.get(self.url+"/state")

            self.json_data = json.loads(response.text)
            self.data_change = not (self.last_data == response.text)
            self.last_data = response.text;
            if self.data_change:
                print(response.text)
            self.connected = True
        except:
            self.connected = False
        return self.data_change 
    
    def send_action(self, action):
        success = True
        try:
            response = requests.get(self.url+"/move/"+action)
            print(f"***send_action: {action}")
            #self.json_data = json.loads(response.text)
            self.connected = True
        except:
            success = False
            self.connected = False
        return success
    
    def get_number_of_players(self):
        num = len(self.json_data["players"])
        return num
    
    def get_name(self,player_num):
        return self.json_data["players"][player_num]["name"]
        
    def get_hand(self,player_num):
        return self.json_data['players'][player_num]['hand']
    
    def get_purse(self,player_num):
        return self.json_data['players'][player_num]['purse']
    
    def get_bet(self,player_num):
        return self.json_data['players'][player_num]['bet']
    
    def get_playing(self, player_num):
        player = self.json_data['activePlayer']
        return player
    
    def get_fold(self, player_num):
        return self.json_data['players'][player_num]['hand'] == "??"
    
    def get_round(self):
        return self.json_data['round']
    
    def get_pot(self):
        return self.json_data['pot']
    
    def get_last_result(self):
        return self.json_data['lastResult']
    
    def get_active_player(self):
        return self.json_data['activePlayer']
    
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
    
    
