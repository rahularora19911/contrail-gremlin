def globals = [:]

globals << [g : graph.traversal(), n : graph.traversal().withStrategies(SubgraphStrategy.build().vertices(hasNot('_missing').hasNot('_incomplete').has('deleted', 0)).create())]
