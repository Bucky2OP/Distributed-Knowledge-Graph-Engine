import requests
import networkx as nx
from typing import Dict, List, Tuple
import sys
import json

class GraphAnalyzer:
    def __init__(self, store_url: str = "http://store:8080"):
        self.store_url = store_url
        self.graph = nx.DiGraph()
    
    def fetch_graph(self) -> bool:
        """Fetch graph data from the store service."""
        try:
            response = requests.get(f"{self.store_url}/export", timeout=10)
            response.raise_for_status()
            data = response.json()
            
            # Build graph
            for node_id, node_data in data["nodes"].items():
                props = node_data.get("props", {})
                self.graph.add_node(node_id, **props)
            
            for edge in data["edges"]:
                self.graph.add_edge(
                    edge["From"], 
                    edge["To"], 
                    label=edge.get("Label", "")
                )
            
            print(f"✓ Successfully loaded graph from {self.store_url}")
            return True
            
        except requests.exceptions.ConnectionError:
            print(f"✗ Error: Cannot connect to {self.store_url}")
            print("  Make sure the graph store service is running")
            return False
        except requests.exceptions.Timeout:
            print(f"✗ Error: Request to {self.store_url} timed out")
            return False
        except Exception as e:
            print(f"✗ Error fetching graph: {e}")
            return False
    
    def basic_stats(self) -> Dict:
        """Calculate basic graph statistics."""
        stats = {
            "nodes": self.graph.number_of_nodes(),
            "edges": self.graph.number_of_edges(),
            "density": nx.density(self.graph),
            "is_connected": nx.is_weakly_connected(self.graph) if self.graph.number_of_nodes() > 0 else False
        }
        return stats
    
    def centrality_analysis(self) -> Dict[str, List[Tuple[str, float]]]:
        """Analyze node centrality using multiple metrics."""
        if self.graph.number_of_nodes() == 0:
            return {}
        
        centrality = {}
        
        # Degree centrality
        degree_cent = nx.degree_centrality(self.graph)
        centrality["degree"] = sorted(degree_cent.items(), key=lambda x: -x[1])[:10]
        
        # In-degree and out-degree
        in_deg = dict(self.graph.in_degree())
        out_deg = dict(self.graph.out_degree())
        centrality["in_degree"] = sorted(in_deg.items(), key=lambda x: -x[1])[:10]
        centrality["out_degree"] = sorted(out_deg.items(), key=lambda x: -x[1])[:10]
        
        # Betweenness centrality (if graph is not too large)
        if self.graph.number_of_nodes() < 1000:
            between_cent = nx.betweenness_centrality(self.graph)
            centrality["betweenness"] = sorted(between_cent.items(), key=lambda x: -x[1])[:10]
        
        # PageRank
        try:
            pagerank = nx.pagerank(self.graph)
            centrality["pagerank"] = sorted(pagerank.items(), key=lambda x: -x[1])[:10]
        except:
            pass
        
        return centrality
    
    def community_detection(self) -> List[set]:
        """Detect communities in the graph."""
        if self.graph.number_of_nodes() == 0:
            return []
        
        # Convert to undirected for community detection
        undirected = self.graph.to_undirected()
        
        try:
            communities = list(nx.community.greedy_modularity_communities(undirected))
            return communities
        except:
            return []
    
    def find_paths(self, source: str, target: str, k: int = 5) -> List[List[str]]:
        """Find shortest paths between two nodes."""
        try:
            paths = list(nx.all_shortest_paths(self.graph, source, target))
            return paths[:k]
        except nx.NetworkXNoPath:
            return []
        except nx.NodeNotFound:
            return []
    
    def export_analysis(self, filename: str = "graph_analysis.json"):
        """Export analysis results to JSON file."""
        stats = self.basic_stats()
        centrality = self.centrality_analysis()
        communities = self.community_detection()
        
        analysis = {
            "statistics": stats,
            "centrality": {
                key: [(node, float(val)) for node, val in values]
                for key, values in centrality.items()
            },
            "communities": [list(comm) for comm in communities],
            "community_count": len(communities)
        }
        
        with open(filename, 'w') as f:
            json.dump(analysis, f, indent=2)
        
        print(f"✓ Analysis exported to {filename}")
    
    def print_report(self):
        """Print a comprehensive analysis report."""
        print("\n" + "="*60)
        print("KNOWLEDGE GRAPH ANALYSIS REPORT")
        print("="*60)
        
        # Basic stats
        stats = self.basic_stats()
        print(f"\n📊 Basic Statistics:")
        print(f"  Nodes: {stats['nodes']}")
        print(f"  Edges: {stats['edges']}")
        print(f"  Density: {stats['density']:.4f}")
        print(f"  Weakly Connected: {stats['is_connected']}")
        
        if stats['nodes'] == 0:
            print("\n⚠️  Graph is empty")
            return
        
        # Centrality
        centrality = self.centrality_analysis()
        
        if "degree" in centrality:
            print(f"\n🎯 Top 10 Nodes by Degree Centrality:")
            for i, (node, score) in enumerate(centrality["degree"], 1):
                print(f"  {i}. {node}: {score:.4f}")
        
        if "pagerank" in centrality:
            print(f"\n📈 Top 10 Nodes by PageRank:")
            for i, (node, score) in enumerate(centrality["pagerank"], 1):
                print(f"  {i}. {node}: {score:.4f}")
        
        # Communities
        communities = self.community_detection()
        if communities:
            print(f"\n🔗 Community Detection:")
            print(f"  Found {len(communities)} communities")
            for i, comm in enumerate(communities[:5], 1):
                print(f"  Community {i}: {len(comm)} nodes")
        
        print("\n" + "="*60 + "\n")


def main():
    # Allow custom store URL via command line
    store_url = sys.argv[1] if len(sys.argv) > 1 else "http://store:8080"
    
    analyzer = GraphAnalyzer(store_url)
    
    if not analyzer.fetch_graph():
        sys.exit(1)
    
    analyzer.print_report()
    analyzer.export_analysis()


if __name__ == "__main__":
    main()