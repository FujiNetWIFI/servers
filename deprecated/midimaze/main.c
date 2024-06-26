#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define BUF_SIZE 1024

struct Client
{
    uint16_t client_num;
    struct sockaddr_in client_addr;
    uint16_t last_packet_seq;
};

struct Client* find_client_by_address(struct Client clients[], int num_clients, const struct sockaddr_in* addr)
{
    for (int i = 0; i < num_clients; i++)
    {
        if (addr->sin_family == clients[i].client_addr.sin_family &&
            addr->sin_port == clients[i].client_addr.sin_port &&
            addr->sin_addr.s_addr == clients[i].client_addr.sin_addr.s_addr)
        {
            return &clients[i];
        }
    }
    printf("Could not find client with address %s:%d\n", inet_ntoa(addr->sin_addr), ntohs(addr->sin_port));
    return NULL;
}

void util_dump_bytes(const uint8_t *buff, uint32_t buff_size)
{
    int bytes_per_line = 16;
    for (int j = 0; j < buff_size; j += bytes_per_line)
    {
        for (int k = 0; (k + j) < buff_size && k < bytes_per_line; k++)
            printf("%02X ", buff[k + j]);
    }
}

int main(int argc, char *argv[])
{
    if (argc != 3)
    {
        fprintf(stderr, "Usage: %s <port> <number of clients>\n", argv[0]);
        return 1;
    }

    int port = atoi(argv[1]);
    int numclients = atoi(argv[2]);

    int sockfd;
    if ((sockfd = socket(AF_INET, SOCK_DGRAM, 0)) < 0)
    {
        perror("socket creation failed");
        return 1;
    }

    struct sockaddr_in servaddr;
    memset(&servaddr, 0, sizeof(servaddr));

    servaddr.sin_family = AF_INET;
    servaddr.sin_addr.s_addr = htonl(INADDR_ANY);
    servaddr.sin_port = htons(port);

    if (bind(sockfd, (struct sockaddr*)&servaddr, sizeof(servaddr)) < 0)
    {
        perror("bind failed");
        return 1;
    }

    struct Client *clients = calloc(numclients, sizeof(struct Client));
    if (!clients)
    {
        perror("Failed to allocate memory for clients");
        return 1;
    }

    int i, nclients = 0;
    while (nclients < numclients)
    {
        char buf[BUF_SIZE];
        memset(buf, 0, BUF_SIZE);

        struct sockaddr_in cliaddr;
        socklen_t clilen = sizeof(cliaddr);

        int recvfrom_ret = recvfrom(sockfd, buf, BUF_SIZE, 0, (struct sockaddr*)&cliaddr, &clilen);
        if (recvfrom_ret < 0)
        {
            perror("recvfrom failed");
            continue;
        }

        if (strncmp(buf, "REGISTER", 8) == 0)
        {
            // Add the client to the clients array
            printf("Client %d registered.\n", nclients+1);
            clients[nclients].client_num = nclients;
            clients[nclients].client_addr = cliaddr;
            clients[nclients].last_packet_seq = 0;
            nclients++;
        }
        else
        {
            printf("Unknown message received: %s\n", buf);
        }
    }

    printf("All clients registered.\n");

    // Loop to forward messages
    while (1)
    {
        char buf[BUF_SIZE];
        memset(buf, 0, BUF_SIZE);

        struct sockaddr_in cliaddr;
        socklen_t clilen = sizeof(cliaddr);

        int recvfrom_ret = recvfrom(sockfd, buf, BUF_SIZE, 0, (struct sockaddr*)&cliaddr, &clilen);
        if (recvfrom_ret < 0)
        {
            perror("recvfrom failed");
            continue;
        }

        struct Client* c = find_client_by_address(clients, numclients, &cliaddr); // Get the active client
        if (c == NULL)
        {
            perror("find_client_by_address null");
            continue;
        }

        uint16_t seq_num = *(uint16_t*)buf;  // Get the sequence number from the packet

        // Check if the packet is in sequence
        if (seq_num <= c->last_packet_seq)
        {
            printf("Ignoring packet %u from client %d, last was %d\n", seq_num, c->client_num, c->last_packet_seq);
            continue;
        }
        else
        {
            // Update client's last sent packet sequence number
            clients[c->client_num].last_packet_seq = seq_num;

            // Remove sequence number from the packet
            memmove(buf, buf+sizeof(uint16_t), BUF_SIZE-sizeof(uint16_t));

            // Forward the message
            int sendto_ret;
            if (c->client_num+1 >= numclients)
            {
                // Back to the first client in the ring
                printf("Packet #%u from address %s:%d to %s:%d\n", seq_num, inet_ntoa(cliaddr.sin_addr), ntohs(cliaddr.sin_port),
                    inet_ntoa(clients[0].client_addr.sin_addr), ntohs(clients[0].client_addr.sin_port));
                sendto_ret = sendto(sockfd, buf, recvfrom_ret - sizeof(uint16_t), 0, (struct sockaddr*)&(clients[0].client_addr), clilen);
            }
            else
            {
                // Next client in the ring
                printf("Packet #%u from address %s:%d to %s:%d\n", seq_num, inet_ntoa(cliaddr.sin_addr), ntohs(cliaddr.sin_port),
                    inet_ntoa(clients[c->client_num+1].client_addr.sin_addr), ntohs(clients[c->client_num+1].client_addr.sin_port));
                sendto_ret = sendto(sockfd, buf, recvfrom_ret - sizeof(uint16_t), 0, (struct sockaddr*)&(clients[c->client_num+1].client_addr), clilen);
            }

            if (sendto_ret < 0) {
                perror("sendto failed");
                continue;
            }
            else
            {
                printf("PKT: ");
                util_dump_bytes(buf, recvfrom_ret - sizeof(uint16_t));
                printf("\n");
            }
        }
    }

    return 0;
}

