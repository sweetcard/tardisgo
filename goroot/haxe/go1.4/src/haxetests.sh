# run the working unit tests using the fastest/most accurate method: C++, JS or neko/interp
for onelevel in errors path
do
	echo "========================================="
	echo "Unit Test (via interpreter): " $onelevel 
	echo "========================================="
	cd $onelevel
	tardisgo -haxe interp -test $onelevel
	if [ "$?" != "0" ]; then
		cd .. 
		exit $?
	fi
	cd .. 
done
for twolevels in container/heap container/list encoding/ascii85 encoding/base32 image/color text/tabwriter unicode/utf8 unicode/utf16 
do
	echo "========================================="
	echo "Unit Test (via interpreter): " $twolevels 
	echo "========================================="
	cd $twolevels
	tardisgo -haxe interp -test $twolevels
	if [ "$?" != "0" ]; then
		cd ../.. 
		exit $?
	fi
	cd ../.. 
done
for onelevel in bufio flag fmt math sort strings unicode 
do
	echo "========================================="
	echo "Unit Test (via js): " $onelevel 
	echo "========================================="
	cd $onelevel
	tardisgo -haxe js -test $onelevel
	if [ "$?" != "0" ]; then
		cd .. 
		exit $?
	fi
	cd .. 
done
for twolevels in container/ring encoding/base64 encoding/csv encoding/hex hash/adler32 hash/crc32 hash/crc64 hash/fnv math/cmplx text/scanner
do
	echo "========================================="
	echo "Unit Test (via js): " $twolevels 
	echo "========================================="
	cd $twolevels
	tardisgo -haxe js -test $twolevels
	if [ "$?" != "0" ]; then
		cd ../.. 
		exit $?
	fi
	cd ../.. 
done
for onelevel in bytes runtime strconv
do
	echo "========================================="
	echo "Unit Test (via C++): " $onelevel 
	echo "========================================="
	cd $onelevel
	tardisgo -haxe cpp -test $onelevel
	if [ "$?" != "0" ]; then
		cd .. 
		exit $?
	fi
	cd .. 
done
for twolevels in image/draw regexp/syntax sync/atomic
do
	echo "========================================="
	echo "Unit Test (via C++): " $twolevels 
	echo "========================================="
	cd $twolevels
	tardisgo -haxe cpp -test $twolevels
	if [ "$?" != "0" ]; then
		cd ../.. 
		exit $?
	fi
	cd ../.. 
done
echo "====================="
echo "All Unit Tests Passed" 
echo "====================="