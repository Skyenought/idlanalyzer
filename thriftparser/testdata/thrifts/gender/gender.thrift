namespace go abcoder.testdata.thrifts.gender
namespace java abcoder.testdata.thrifts.gender

include "../common/entity/entity.thrift"

struct Gender {
	1: string gender
	2: entity.Entity e
}