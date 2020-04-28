#

default: test1.png

test1.png: pngup.rb README.md
	./pngup.rb
	mv tmp.png $@
